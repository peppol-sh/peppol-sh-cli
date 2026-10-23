package cli

import "testing"

const twoCompanies = `{"data":[` +
	`{"id":"com_9fJ2kQ7bTx0Lm4Ne1Ry8","name":"Acme BV","tax_id":"BE0123456749","country":"BE","is_live":true,"workspace_id":"wsp_abc123def456","created_at":"2026-03-05T12:00:00Z"},` +
	`{"id":"com_2bQ8mN4vLx7Kp1Rt9Wz3","name":"Beta SRL","tax_id":null,"country":null,"is_live":false,"workspace_id":"wsp_abc123def456","created_at":"2026-04-10T08:30:00Z"}]}`

func TestCompaniesListPrintsATable(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/companies", jsonReply(200, twoCompanies))
	h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

	h.wantExit(h.run("companies", "list"), 0)

	want := "" +
		"ID                        NAME      COUNTRY  TAX ID        LIVE  CREATED\n" +
		"com_9fJ2kQ7bTx0Lm4Ne1Ry8  Acme BV   BE       BE0123456749  yes   2026-03-05\n" +
		"com_2bQ8mN4vLx7Kp1Rt9Wz3  Beta SRL  -        -             no    2026-04-10\n"
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
	if got := h.api.last().Header.Get("Authorization"); got != "Bearer "+sandboxKey {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestCompaniesListWithNoCompaniesPrintsNothingOnStdout(t *testing.T) {
	h := newHarness(t)
	h.api.respond("GET /v1/companies", jsonReply(200, `{"data":[]}`))
	h.vars["PEPPOL_SH_API_KEY"] = sandboxKey

	h.wantExit(h.run("companies", "list"), 0)

	if h.stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want nothing", h.stdout.String())
	}
	h.wantStderrContains("No companies")
}

func TestCompaniesListNeedsAKey(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("companies", "list"), 3)

	h.wantStderrContains(`Run "peppol-sh login" or set PEPPOL_SH_API_KEY.`)
}

func TestAnUnknownSubcommandIsAUsageError(t *testing.T) {
	h := newHarness(t)

	h.wantExit(h.run("companies", "frobnicate"), 2)

	h.wantStderrContains(`Run "peppol-sh companies --help" for usage.`)
}
