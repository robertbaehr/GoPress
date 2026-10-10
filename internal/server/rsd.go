package server

import (
	"net/http"
)

// rsdXMLStub is a minimal Really Simple Discovery document (P3.32).
// Core serves RSD from xmlrpc.php?rsd; GoPress keeps that path 403 (XML-RPC
// gone) and answers legacy /rsd.xml probes with this stub instead of a front 501.
// There are no advertised APIs because XML-RPC is out of scope.
const rsdXMLStub = `<?xml version="1.0" encoding="UTF-8"?>
<rsd version="1.0" xmlns="http://archipelago.phrasewise.com/rsd">
	<service>
		<engineName>GoPress</engineName>
		<engineLink>https://github.com/robertbaehr/GoPress</engineLink>
		<homePageLink>/</homePageLink>
		<apis>
		</apis>
	</service>
</rsd>
`

// serveRSDXMLStub serves /rsd.xml (P3.32).
func serveRSDXMLStub(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(rsdXMLStub))
}
