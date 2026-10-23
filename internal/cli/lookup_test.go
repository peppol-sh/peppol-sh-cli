package cli

import "testing"

const lookupResult = `{"participant_id":{"scheme":"0208","id":"0123456749"},"domain":"sml",` +
	`"naptr_domain":"B-1f3870be274f6c49b3e31a0c6728957f.iso6523-actorid-upis.edelivery.tech.ec.europa.eu",` +
	`"smp_url":"https://smp.example.com","services":[` +
	`{"document_type_id":"busdox-docid-qns::urn:oasis:names:specification:ubl:schema:xsd:Invoice-2::Invoice","document_type_name":"Peppol BIS Billing UBL Invoice V3",` +
	`"process_id":"cenbii-procid-ubl::urn:fdc:peppol.eu:2017:poacc:billing:01:1.0","process_name":"Peppol BIS Billing 3.0","endpoints":[]},` +
	`{"document_type_id":"urn:example:creditnote","document_type_name":null,"process_id":"urn:example:billing","process_name":null,"endpoints":[]}],` +
	`"business_card":{"participant_id":{"scheme":"iso6523-actorid-upis","id":"0208:0123456749"},"entities":[{"name":"Acme BV","country_code":"BE"}]}}`

func TestLookupPrintsTheParticipantAndItsServices(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(200, lookupResult))

	h.wantExit(h.run("lookup", "0208:0123456749"), 0)

	want := "" +
		"Participant  0208:0123456749\n" +
		"Domain       sml\n" +
		"SMP          https://smp.example.com\n" +
		"Entity       Acme BV (BE)\n" +
		"\n" +
		"PROCESS                 DOCUMENT TYPE\n" +
		"Peppol BIS Billing 3.0  Peppol BIS Billing UBL Invoice V3\n" +
		"urn:example:billing     urn:example:creditnote\n"
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
}

func TestLookupWithoutAKeyCallsTheLiveHostAnonymously(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(200, lookupResult))

	h.wantExit(h.run("lookup", "0208:0123456749"), 0)

	req := h.api.last()
	if got := req.URL.String(); got != "https://api.peppol.sh/v1/lookup/0208:0123456749" {
		t.Fatalf("request URL = %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none", got)
	}
}

func TestLookupSendsTheKeyWhenThereIsOne(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(200, lookupResult))
	h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

	h.wantExit(h.run("lookup", "0208:0123456749"), 0)

	if got := h.api.last().Header.Get("Authorization"); got != "Bearer "+sandboxKey {
		t.Fatalf("Authorization = %q, want the key", got)
	}
}

func TestLookupIgnoresAKeyWithAnUnknownPrefix(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(200, lookupResult))
	h.vars["PEPPOL_SH_API_KEY"] = "junk"

	h.wantExit(h.run("lookup", "0208:0123456749"), 0)

	req := h.api.last()
	if got := req.URL.String(); got != "https://api.peppol.sh/v1/lookup/0208:0123456749" {
		t.Fatalf("request URL = %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none", got)
	}
	h.wantStderrContains("Warning: ignoring the API key from PEPPOL_SH_API_KEY: it does not start with ps_test_ or ps_live_.\n")
}

func TestLookupPassesTheDomain(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(200, lookupResult))

	h.wantExit(h.run("lookup", "0208:0123456749", "--domain", "smk"), 0)

	if got := h.api.last().URL.String(); got != "https://api.peppol.sh/v1/lookup/0208:0123456749?domain=smk" {
		t.Fatalf("request URL = %q", got)
	}
}

func TestLookupRejectsAnUnknownDomain(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("lookup", "0208:0123456749", "--domain", "prod"), 2)

	if n := h.api.count(); n != 0 {
		t.Fatalf("sent %d requests, want 0", n)
	}
}

func TestLookupNeedsOneParticipant(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("lookup"), 2)
}

func TestLookupErrorsExit1(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantStderr string
	}{
		{
			name:       "not found",
			status:     404,
			body:       `{"error":{"type":"not_found","code":"participant_not_found","message":"Participant not found in SML"}}`,
			wantStderr: "Error: participant_not_found: Participant not found in SML (HTTP 404)\n",
		},
		{
			name:       "bad id",
			status:     400,
			body:       `{"error":{"type":"validation_error","code":"invalid_scheme","message":"Unknown EAS scheme \"9999\""}}`,
			wantStderr: "Error: invalid_scheme: Unknown EAS scheme \"9999\" (HTTP 400)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.api.respond("GET /v1/lookup/{peppol_id}", jsonReply(tt.status, tt.body))

			h.wantExit(h.run("lookup", "9999:0123456749"), 1)

			if got := h.stderr.String(); got != tt.wantStderr {
				t.Fatalf("stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}
