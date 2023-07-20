#/bin/sh
curl -k -vvv -H "Content-Type: application/json" --data '{"using":["urn:ietf:params:jmap:core","urn:ietf:params:jmap:mail"],"methodCalls":[["Mailbox/get",{"accountId":"000","ids":null},"0"]]}' https://mox%40localhost:moxmoxmox@localhost:1443/jmap/api 
