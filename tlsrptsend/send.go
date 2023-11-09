// Package tlsrptsend sends TLS reports based on success/failure statistics and
// details gathering while making SMTP STARTTLS connections for delivery. See RFC
// 8460.
package tlsrptsend

// tlsrptsend is a separate package instead of being in tlsrptdb because it imports
// queue and queue imports tlsrptdb to store tls results, so that would cause a
// cyclic dependency.

// Sending TLS reports and DMARC reports is very similar. See ../dmarcdb/eval.go:/similar and ../tlsrptsend/send.go:/similar.

// todo spec: ../rfc/8460:441 ../rfc/8460:463 may lead reader to believe they can find a DANE or MTA-STS policy at the same place, while in practice you'll get an MTA-STS policy at a recipient domain and a DANE policy at a mail host, and that's where the TLSRPT policy is defined. it would have helped with this implementation if the distinction was mentioned explicitly, also earlier in the document (i realized it late in the implementation process based on the terminology entry for the policy domain). examples with a tlsrpt record at a mail host would have helped too.
// todo spec: ../rfc/8460:1017 example report message misses the required DKIM signature.

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/mjl-/bstore"

	"github.com/mjl-/mox/config"
	"github.com/mjl-/mox/dkim"
	"github.com/mjl-/mox/dns"
	"github.com/mjl-/mox/message"
	"github.com/mjl-/mox/metrics"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/mox-"
	"github.com/mjl-/mox/moxio"
	"github.com/mjl-/mox/moxvar"
	"github.com/mjl-/mox/queue"
	"github.com/mjl-/mox/smtp"
	"github.com/mjl-/mox/store"
	"github.com/mjl-/mox/tlsrpt"
	"github.com/mjl-/mox/tlsrptdb"
)

var (
	metricReport = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mox_tlsrptsend_report_queued_total",
			Help: "Total messages with TLS reports queued.",
		},
	)
	metricReportError = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mox_tlsrptsend_report_error_total",
			Help: "Total errors while composing or queueing TLS reports.",
		},
	)
)

var jitterRand = mox.NewPseudoRand()

// time to sleep until sending reports at midnight t, replaced by tests.
// Jitter so we don't cause load at exactly midnight, other processes may
// already be doing that.
var jitteredTimeUntil = func(t time.Time) time.Duration {
	return time.Until(t.Add(time.Duration(240+jitterRand.Intn(120)) * time.Second))
}

// Start launches a goroutine that wakes up just after 00:00 UTC to send TLSRPT
// reports. Reports are sent spread out over a 4 hour period.
func Start(resolver dns.Resolver) {
	go func() {
		log := mlog.New("tlsrptsend")

		defer func() {
			// In case of panic don't take the whole program down.
			x := recover()
			if x != nil {
				log.Error("recover from panic", mlog.Field("panic", x))
				debug.PrintStack()
				metrics.PanicInc(metrics.Tlsrptdb)
			}
		}()

		timer := time.NewTimer(time.Hour) // Reset below.
		defer timer.Stop()

		ctx := mox.Shutdown

		db := tlsrptdb.ResultDB
		if db == nil {
			log.Error("no tlsrpt results database for tls reports, not sending reports")
			return
		}

		// We start sending for previous day, if there are any reports left.
		endUTC := midnightUTC(time.Now())

		for {
			dayUTC := endUTC.Add(-12 * time.Hour).Format("20060102")

			// Remove evaluations older than 48 hours (2 reports with 24 hour interval)
			// They should have been processed by now. We may have kept them
			// during temporary errors, but persistent temporary errors shouldn't fill up our
			// database and we don't want to send old reports either.
			_, err := bstore.QueryDB[tlsrptdb.TLSResult](ctx, db).FilterLess("DayUTC", endUTC.Add((-48-12)*time.Hour).Format("20060102")).Delete()
			log.Check(err, "removing stale tls results from database")

			log.Info("sending tls reports", mlog.Field("day", dayUTC))
			if err := sendReports(ctx, log.WithCid(mox.Cid()), resolver, db, dayUTC, endUTC); err != nil {
				log.Errorx("sending tls reports", err)
				metricReportError.Inc()
			} else {
				log.Info("finished sending tls reports")
			}

			endUTC = endUTC.Add(24 * time.Hour)
			timer.Reset(jitteredTimeUntil(endUTC))

			select {
			case <-ctx.Done():
				log.Info("tls report sender shutting down")
				return
			case <-timer.C:
			}
		}
	}()
}

func midnightUTC(now time.Time) time.Time {
	t := now.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Sleep in between sending two reports.
// Replaced by tests.
var sleepBetween = func(ctx context.Context, between time.Duration) (ok bool) {
	t := time.NewTimer(between)
	select {
	case <-ctx.Done():
		t.Stop()
		return false
	case <-t.C:
		return true
	}
}

// sendReports gathers all policy domains that have results that should receive a
// TLS report and sends a report to each if their TLSRPT DNS record has reporting
// addresses.
func sendReports(ctx context.Context, log *mlog.Log, resolver dns.Resolver, db *bstore.DB, dayUTC string, endTimeUTC time.Time) error {
	type key struct {
		policyDomain string
		dayUTC       string
	}
	destDomains := map[key]bool{}

	// Gather all policy domains we plan to send to.
	var nsend int
	q := bstore.QueryDB[tlsrptdb.TLSResult](ctx, db)
	q.FilterLessEqual("DayUTC", dayUTC)
	q.SortAsc("PolicyDomain", "DayUTC", "RecipientDomain") // Sort for testability.
	err := q.ForEach(func(e tlsrptdb.TLSResult) error {
		k := key{e.PolicyDomain, dayUTC}
		if e.SendReport && !destDomains[k] {
			nsend++
		}
		destDomains[k] = destDomains[k] || e.SendReport
		return nil
	})
	if err != nil {
		return fmt.Errorf("looking for domains to send tls reports to: %v", err)
	}

	// Send report to each domain. We stretch sending over 4 hours, but only if there
	// are quite a few message. ../rfc/8460:479
	between := 4 * time.Hour
	if nsend > 0 {
		between = between / time.Duration(nsend)
	}
	if between > 5*time.Minute {
		between = 5 * time.Minute
	}

	var wg sync.WaitGroup

	var n int
	for k, send := range destDomains {
		// Cleanup results for domain that doesn't need to get a report (e.g. for TLS
		// connections that were the result of delivering TLSRPT messages).
		if !send {
			removeResults(ctx, log, db, k.policyDomain, k.dayUTC)
			continue
		}

		if n > 0 {
			ok := sleepBetween(ctx, between)
			if !ok {
				return nil
			}
		}
		n++

		// In goroutine, so our timing stays independent of how fast we process.
		wg.Add(1)
		go func(policyDomain string, dayUTC string) {
			defer func() {
				// In case of panic don't take the whole program down.
				x := recover()
				if x != nil {
					log.Error("unhandled panic in tlsrptsend sendReports", mlog.Field("panic", x))
					debug.PrintStack()
					metrics.PanicInc(metrics.Tlsrptdb)
				}
			}()
			defer wg.Done()

			rlog := log.WithCid(mox.Cid()).Fields(mlog.Field("policydomain", policyDomain), mlog.Field("daytutc", dayUTC))
			if _, err := sendReportDomain(ctx, rlog, resolver, db, endTimeUTC, policyDomain, dayUTC); err != nil {
				rlog.Errorx("sending tls report to domain", err)
				metricReportError.Inc()
			}
		}(k.policyDomain, k.dayUTC)
	}

	wg.Wait()

	return nil
}

func removeResults(ctx context.Context, log *mlog.Log, db *bstore.DB, policyDomain string, dayUTC string) {
	q := bstore.QueryDB[tlsrptdb.TLSResult](ctx, db)
	q.FilterNonzero(tlsrptdb.TLSResult{PolicyDomain: policyDomain, DayUTC: dayUTC})
	_, err := q.Delete()
	log.Check(err, "removing tls results from database")
}

// replaceable for testing.
var queueAdd = queue.Add

func sendReportDomain(ctx context.Context, log *mlog.Log, resolver dns.Resolver, db *bstore.DB, endUTC time.Time, policyDomain, dayUTC string) (cleanup bool, rerr error) {
	dom, err := dns.ParseDomain(policyDomain)
	if err != nil {
		return false, fmt.Errorf("parsing policy domain for sending tls reports: %v", err)
	}

	// We'll cleanup records by default.
	cleanup = true
	// But if we encounter a temporary error we cancel cleanup of evaluations on error.
	tempError := false

	defer func() {
		if !cleanup || tempError {
			log.Debug("not cleaning up results after attempting to send tls report")
		} else {
			removeResults(ctx, log, db, policyDomain, dayUTC)
		}
	}()

	// Get TLSRPT record. If there are no reporting addresses, we're not going to send at all.
	record, _, err := tlsrpt.Lookup(ctx, resolver, dom)
	if err != nil {
		// If there is no TLSRPT record, that's fine, we'll remove what we tracked.
		if errors.Is(err, tlsrpt.ErrNoRecord) {
			return true, nil
		}
		cleanup = errors.Is(err, tlsrpt.ErrDNS)
		return cleanup, fmt.Errorf("looking up current tlsrpt record for reporting addresses: %v", err)
	}

	var recipients []message.NameAddress

	for _, l := range record.RUAs {
		for _, s := range l {
			u, err := url.Parse(s)
			if err != nil {
				log.Debugx("parsing rua uri in tlsrpt dns record, ignoring", err, mlog.Field("rua", s))
				continue
			}

			if u.Scheme == "mailto" {
				addr, err := smtp.ParseAddress(u.Opaque)
				if err != nil {
					log.Debugx("parsing mailto uri in tlsrpt record rua value, ignoring", err, mlog.Field("rua", s))
					continue
				}
				recipients = append(recipients, message.NameAddress{Address: addr})
			} else if u.Scheme == "https" {
				// Although "report" is ambiguous and could mean both only the JSON data or an
				// entire message (including DKIM-Signature) with the JSON data, it appears the
				// intention of the RFC is that the HTTPS transport sends only the JSON data, given
				// mention of the media type to use (for the HTTP POST). It is the type of the
				// report, not of a message. TLS reports sent over email must have a DKIM
				// signature, i.e. must be authenticated, for understandable reasons. No such
				// requirement is specified for HTTPS, but no one is going to accept
				// unauthenticated TLS reports over HTTPS. So there seems little point in sending
				// them.
				// ../rfc/8460:320 ../rfc/8460:1055
				// todo spec: would be good to have clearer distinction between "report" (JSON) and "report message" (message with report attachment, that can be DKIM signed). propose sending report message over https that includes DKIM signature so authenticity can be verified and the report used. ../rfc/8460:310
				log.Debug("https scheme in rua uri in tlsrpt record, ignoring since they will likey not be used to due lack of authentication", mlog.Field("rua", s))
			} else {
				log.Debug("unknown scheme in rua uri in tlsrpt record, ignoring", mlog.Field("rua", s))
			}
		}
	}

	if len(recipients) == 0 {
		// No reports requested, perfectly fine, no work to do for us.
		log.Debug("no tlsrpt reporting addresses configured")
		return true, nil
	}

	log.Info("sending tlsrpt report")

	q := bstore.QueryDB[tlsrptdb.TLSResult](ctx, db)
	q.FilterNonzero(tlsrptdb.TLSResult{PolicyDomain: policyDomain, DayUTC: dayUTC})
	tlsResults, err := q.List()
	if err != nil {
		return true, fmt.Errorf("get tls results from database: %v", err)
	}

	if len(tlsResults) == 0 {
		// Should not happen. But no point in sending messages with empty reports.
		return true, fmt.Errorf("no tls results found")
	}

	beginUTC := endUTC.Add(-24 * time.Hour)

	report := tlsrpt.Report{
		OrganizationName: mox.Conf.Static.HostnameDomain.ASCII,
		DateRange: tlsrpt.TLSRPTDateRange{
			Start: beginUTC,
			End:   endUTC.Add(-time.Second), // Per example, ../rfc/8460:1769
		},
		ContactInfo: "postmaster@" + mox.Conf.Static.HostnameDomain.ASCII,
		// todo spec: ../rfc/8460:968 ../rfc/8460:1772 ../rfc/8460:691 subject header assumes a report-id in the form of a msg-id, but example and report-id json field explanation allows free-form report-id's (assuming we're talking about the same report-id here).
		ReportID: endUTC.Format("20060102") + "." + dom.ASCII + "@" + mox.Conf.Static.HostnameDomain.ASCII,
	}

	// Merge all results into this report.
	for _, tlsResult := range tlsResults {
		report.Merge(tlsResult.Results...)
	}

	reportFile, err := store.CreateMessageTemp("tlsreportout")
	if err != nil {
		return false, fmt.Errorf("creating temporary file for outgoing tls report: %v", err)
	}
	defer store.CloseRemoveTempFile(log, reportFile, "generated tls report")

	// ../rfc/8460:905
	gzw := gzip.NewWriter(reportFile)
	enc := json.NewEncoder(gzw)
	enc.SetIndent("", "\t")
	if err == nil {
		err = enc.Encode(report)
	}
	if err == nil {
		err = gzw.Close()
	}
	if err != nil {
		return false, fmt.Errorf("writing tls report as json with gzip: %v", err)
	}

	msgf, err := store.CreateMessageTemp("tlsreportmsgout")
	if err != nil {
		return false, fmt.Errorf("creating temporary message file with outgoing tls report: %v", err)
	}
	defer store.CloseRemoveTempFile(log, msgf, "message with generated tls report")

	// We are sending reports from our host's postmaster address. In a
	// typical setup the host is a subdomain of a configured domain with
	// DKIM keys, so we can DKIM-sign our reports. SPF should pass anyway.
	// todo future: when sending, use an SMTP MAIL FROM that we can relate back to recipient reporting address so we can stop trying to send reports in case of repeated delivery failure DSNs.
	from := smtp.Address{Localpart: "postmaster", Domain: mox.Conf.Static.HostnameDomain}

	// Subject follows the form from RFC. ../rfc/8460:959
	subject := fmt.Sprintf("Report Domain: %s Submitter: %s Report-ID: <%s>", dom.ASCII, mox.Conf.Static.HostnameDomain.ASCII, report.ReportID)

	// Human-readable part for convenience. ../rfc/8460:917
	text := fmt.Sprintf(`
Attached is a TLS report with a summary of connection successes and failures
during attempts to securely deliver messages to your mail server, including
details about errors encountered. You are receiving this message because your
address is specified in the "rua" field of the TLSRPT record for your
domain/host.

Policy Domain: %s
Submitter: %s
Report-ID: %s
Period: %s - %s UTC
`, dom, mox.Conf.Static.HostnameDomain, report.ReportID, beginUTC.Format(time.DateTime), endUTC.Format(time.DateTime))

	// The attached file follows the naming convention from the RFC. ../rfc/8460:849
	reportFilename := fmt.Sprintf("%s!%s!%d!%d.json.gz", mox.Conf.Static.HostnameDomain.ASCII, dom.ASCII, beginUTC.Unix(), endUTC.Add(-time.Second).Unix())

	// Compose the message.
	msgPrefix, has8bit, smtputf8, messageID, err := composeMessage(ctx, log, msgf, dom, from, recipients, subject, text, reportFilename, reportFile)
	if err != nil {
		return false, fmt.Errorf("composing message with outgoing tls report: %v", err)
	}
	msgInfo, err := msgf.Stat()
	if err != nil {
		return false, fmt.Errorf("stat message with outgoing tls report: %v", err)
	}
	msgSize := int64(len(msgPrefix)) + msgInfo.Size()

	for _, rcpt := range recipients {
		qm := queue.MakeMsg(mox.Conf.Static.Postmaster.Account, from.Path(), rcpt.Address.Path(), has8bit, smtputf8, msgSize, messageID, []byte(msgPrefix), nil)
		// Don't try as long as regular deliveries, and stop before we would send the
		// delayed DSN. Though we also won't send that due to IsTLSReport.
		// ../rfc/8460:1077
		qm.MaxAttempts = 5
		qm.IsTLSReport = true
		// TLS failures should be ignored. ../rfc/8460:317 ../rfc/8460:1050
		no := false
		qm.RequireTLS = &no

		err := queueAdd(ctx, log, &qm, msgf)
		if err != nil {
			tempError = true
			log.Errorx("queueing message with tls report", err)
			metricReportError.Inc()
		} else {
			log.Debug("tls report queued", mlog.Field("recipient", rcpt))
			metricReport.Inc()
		}
	}

	// Regardless of whether we queued a report, we are not going to keep the
	// evaluations around. Though this can be overridden if tempError is set.
	// ../rfc/7489:1785

	return true, nil
}

func composeMessage(ctx context.Context, log *mlog.Log, mf *os.File, policyDomain dns.Domain, fromAddr smtp.Address, recipients []message.NameAddress, subject, text, filename string, reportFile *os.File) (msgPrefix string, has8bit, smtputf8 bool, messageID string, rerr error) {
	xc := message.NewComposer(mf, 100*1024*1024)
	defer func() {
		x := recover()
		if x == nil {
			return
		}
		if err, ok := x.(error); ok && errors.Is(err, message.ErrCompose) {
			rerr = err
			return
		}
		panic(x)
	}()

	// We only use smtputf8 if we have to, with a utf-8 localpart. For IDNA, we use ASCII domains.
	for _, a := range recipients {
		if a.Address.Localpart.IsInternational() {
			xc.SMTPUTF8 = true
			break
		}
	}

	xc.HeaderAddrs("From", []message.NameAddress{{Address: fromAddr}})
	xc.HeaderAddrs("To", recipients)
	xc.Subject(subject)
	// ../rfc/8460:926
	xc.Header("TLS-Report-Domain", policyDomain.ASCII)
	xc.Header("TLS-Report-Submitter", mox.Conf.Static.HostnameDomain.ASCII)
	// TLS failures should be ignored. ../rfc/8460:317 ../rfc/8460:1050
	xc.Header("TLS-Required", "No")
	messageID = fmt.Sprintf("<%s>", mox.MessageIDGen(xc.SMTPUTF8))
	xc.Header("Message-Id", messageID)
	xc.Header("Date", time.Now().Format(message.RFC5322Z))
	xc.Header("User-Agent", "mox/"+moxvar.Version)
	xc.Header("MIME-Version", "1.0")

	// Multipart message, with a text/plain and the report attached.
	mp := multipart.NewWriter(xc)
	// ../rfc/8460:916
	xc.Header("Content-Type", fmt.Sprintf(`multipart/report; report-type="tlsrpt"; boundary="%s"`, mp.Boundary()))
	xc.Line()

	// Textual part, just mentioning this is a TLS report.
	textBody, ct, cte := xc.TextPart(text)
	textHdr := textproto.MIMEHeader{}
	textHdr.Set("Content-Type", ct)
	textHdr.Set("Content-Transfer-Encoding", cte)
	textp, err := mp.CreatePart(textHdr)
	xc.Checkf(err, "adding text part to message")
	_, err = textp.Write(textBody)
	xc.Checkf(err, "writing text part")

	// TLS report as attachment.
	ahdr := textproto.MIMEHeader{}
	ct = mime.FormatMediaType("application/tlsrpt+gzip", map[string]string{"name": filename})
	ahdr.Set("Content-Type", ct)
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	ahdr.Set("Content-Disposition", cd)
	ahdr.Set("Content-Transfer-Encoding", "base64")
	ap, err := mp.CreatePart(ahdr)
	xc.Checkf(err, "adding tls report to message")
	wc := moxio.Base64Writer(ap)
	_, err = io.Copy(wc, &moxio.AtReader{R: reportFile})
	xc.Checkf(err, "adding attachment")
	err = wc.Close()
	xc.Checkf(err, "flushing attachment")

	err = mp.Close()
	xc.Checkf(err, "closing multipart")

	xc.Flush()

	// Also sign the TLS-Report headers. ../rfc/8460:940
	extraHeaders := []string{"TLS-Report-Domain", "TLS-Report-Submitter"}
	msgPrefix = dkimSign(ctx, log, fromAddr, smtputf8, mf, extraHeaders)

	return msgPrefix, xc.Has8bit, xc.SMTPUTF8, messageID, nil
}

func dkimSign(ctx context.Context, log *mlog.Log, fromAddr smtp.Address, smtputf8 bool, mf *os.File, extraHeaders []string) string {
	// Add DKIM-Signature headers if we have a key for (a higher) domain than the from
	// address, which is a host name. A signature will only be useful with higher-level
	// domains if they have a relaxed dkim check (which is the default). If the dkim
	// check is strict, there is no harm, there will simply not be a dkim pass.
	fd := fromAddr.Domain
	var zerodom dns.Domain
	for fd != zerodom {
		confDom, ok := mox.Conf.Domain(fd)
		if ok && len(confDom.DKIM.Sign) == 0 {
			return ""
		}
		if len(confDom.DKIM.Sign) > 0 {
			selectors := map[string]config.Selector{}
			for name, sel := range confDom.DKIM.Selectors {
				sel.HeadersEffective = append(append([]string{}, sel.HeadersEffective...), extraHeaders...)
				selectors[name] = sel
			}
			confDom.DKIM.Selectors = selectors

			dkimHeaders, err := dkim.Sign(ctx, fromAddr.Localpart, fd, confDom.DKIM, smtputf8, mf)
			if err != nil {
				log.Errorx("dkim-signing dmarc report, continuing without signature", err)
				metricReportError.Inc()
				return ""
			}
			return dkimHeaders
		}

		var nfd dns.Domain
		_, nfd.ASCII, _ = strings.Cut(fd.ASCII, ".")
		_, nfd.Unicode, _ = strings.Cut(fd.Unicode, ".")
		fd = nfd
	}
	return ""
}
