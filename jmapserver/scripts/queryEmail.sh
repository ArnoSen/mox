#/bin/sh

curl -k -vvv -H "Content-Type: application/json" --data '{"using":["urn:ietf:params:jmap:core","urn:ietf:params:jmap:mail"],"methodCalls":[["Email/query",{"accountId":"000","calculateTotal":true,"collapseThreads":true, "filter":{"inMailbox":"1"},"limit":30,"position":0,"sort": [{"isAscending":false, "property":"receivedAt"}]},"0"]]}' https://mox%40localhost:moxmoxmox@localhost:1443/jmap/api 
