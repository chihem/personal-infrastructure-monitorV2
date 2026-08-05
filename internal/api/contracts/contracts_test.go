package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestMonitoringExamplesValidateAndRoundTrip(t *testing.T) {
	fixtures := []string{
		"snapshot-complete.json",
		"snapshot-partial.json",
		"snapshot-stale.json",
		"snapshot-unavailable.json",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			original := readFixture(t, fixture)
			var first Envelope[MonitoringSnapshot]
			if err := json.Unmarshal(original, &first); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			if err := ValidateMonitoringEnvelope(first); err != nil {
				t.Fatalf("validate fixture: %v", err)
			}

			encoded, err := json.Marshal(first)
			if err != nil {
				t.Fatalf("marshal contract: %v", err)
			}
			var second Envelope[MonitoringSnapshot]
			if err := json.Unmarshal(encoded, &second); err != nil {
				t.Fatalf("unmarshal round trip: %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("JSON round trip changed the contract value")
			}
		})
	}
}

func TestChartExampleDistinguishesUnavailablePointAndGap(t *testing.T) {
	var series ChartSeries
	if err := json.Unmarshal(readFixture(t, "chart-with-gap.json"), &series); err != nil {
		t.Fatalf("unmarshal chart fixture: %v", err)
	}
	if err := series.Validate(); err != nil {
		t.Fatalf("validate chart fixture: %v", err)
	}
	if series.Points[1].Measurement == nil || series.Points[1].Measurement.Availability != domain.AvailabilityUnavailable {
		t.Fatal("collected unavailable point was not preserved")
	}
	if series.Points[2].State != ChartGap || series.Points[2].Measurement != nil {
		t.Fatal("chart gap was not preserved")
	}

	measurement := domain.Metric[float64]{
		Availability: domain.AvailabilityAvailable,
		Value:        floatPointer(10),
		Unit:         domain.UnitPercent,
	}
	invalidGap := ChartPoint{
		Timestamp:   time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		State:       ChartGap,
		Measurement: &measurement,
	}
	if err := invalidGap.Validate(); err == nil {
		t.Fatal("gap containing a fabricated measurement was accepted")
	}
}

func TestEnvelopeRequiresDataXorError(t *testing.T) {
	envelope := Envelope[MonitoringSnapshot]{
		APIVersion:  APIVersion,
		RequestID:   "request-001",
		GeneratedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
	if err := envelope.Validate(nil); err == nil {
		t.Fatal("envelope without data or error was accepted")
	}

	data := MonitoringSnapshot{}
	apiError := APIError{
		Code:            domain.ErrorUnavailable,
		MessageKey:      "errors.unavailable",
		FallbackMessage: "The requested information is unavailable.",
		FieldErrors:     []FieldError{},
	}
	envelope.Data = &data
	envelope.Error = &apiError
	if err := envelope.Validate(nil); err == nil {
		t.Fatal("envelope containing both data and error was accepted")
	}
}

func TestEnvelopeRejectsNonUTCTimestamp(t *testing.T) {
	data := MonitoringSnapshot{}
	envelope := Envelope[MonitoringSnapshot]{
		APIVersion:  APIVersion,
		RequestID:   "request-001",
		GeneratedAt: time.Date(2026, 8, 4, 13, 0, 0, 0, time.FixedZone("Africa/Tunis", 3600)),
		Data:        &data,
	}
	if err := envelope.Validate(nil); err == nil {
		t.Fatal("non-UTC API timestamp was accepted")
	}
}

func TestPagingAndConfirmedActionsValidate(t *testing.T) {
	cursor := "next-page"
	if err := (PageInfo{Limit: 50, HasMore: true, NextCursor: &cursor}).Validate(); err != nil {
		t.Fatalf("valid page rejected: %v", err)
	}
	if err := (PageInfo{Limit: 50, HasMore: true}).Validate(); err == nil {
		t.Fatal("page missing its next cursor was accepted")
	}

	request := ConfirmationRequest{
		Action: domain.ActionDockerRestart,
		Target: domain.ResourceRef{
			Kind: domain.ResourceContainer, ID: "container-abc", DisplayName: "example",
		},
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("valid confirmation rejected: %v", err)
	}
	request.Target.Kind = domain.ResourceFilesystem
	if err := request.Validate(); err == nil {
		t.Fatal("Docker action targeting a filesystem was accepted")
	}
}

func TestAuditDeleteScopesAreUnambiguous(t *testing.T) {
	selected := AuditDeleteRequest{Scope: AuditDeleteSelected, IDs: []string{"audit-1"}}
	if err := selected.Validate(); err != nil {
		t.Fatalf("selected deletion rejected: %v", err)
	}
	ambiguous := AuditDeleteRequest{Scope: AuditDeleteAll, IDs: []string{"audit-1"}}
	if err := ambiguous.Validate(); err == nil {
		t.Fatal("all deletion containing selected ids was accepted")
	}
}

func TestExportRangesMatchApprovedPresets(t *testing.T) {
	request := ExportRequest{
		Format:   ExportJSON,
		Datasets: []ExportDataset{ExportCPU},
		Range:    domain.RangeSelection{Preset: domain.RangeLastHour},
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("approved export range rejected: %v", err)
	}
	request.Range.Preset = domain.RangeLast5Minutes
	if err := request.Validate(); err == nil {
		t.Fatal("unapproved five-minute export range was accepted")
	}
}

func TestRequiredCollectionsEncodeAsArrays(t *testing.T) {
	var envelope Envelope[MonitoringSnapshot]
	if err := json.Unmarshal(readFixture(t, "snapshot-complete.json"), &envelope); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	envelope.Data.Filesystems = nil
	if err := ValidateMonitoringEnvelope(envelope); err == nil {
		t.Fatal("nil filesystems collection was accepted")
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tests", "fixtures", "contracts", "v1", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return contents
}

func floatPointer(value float64) *float64 {
	return &value
}
