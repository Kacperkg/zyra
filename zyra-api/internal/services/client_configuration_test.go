package services

import (
	"math"
	"testing"
	"time"
	"zyra-api/internal/models"
)

func TestUniqueIDObservationParsing(t *testing.T) {
	for _, sample := range []struct{ body, want string }{
		{"UniqueID: host-db-instance\r\nDatabase: D", "host-db-instance"},
		{"UniqueID:\nDatabase: D", ""},
		{"Database: D", ""},
		{"not UniqueID: guessed", ""},
	} {
		if got := ParseReport(sample.body, models.CheckSettings{}).UniqueID; got != sample.want {
			t.Fatalf("got %q want %q", got, sample.want)
		}
	}
}

func TestThresholdsRejectNonFinite(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, 101} {
		if ValidateSettings(models.CheckSettings{Resources: map[string]map[string]models.ResourceRule{"filesystem": {"/": {MaxUsedPercent: &value}}}}) == nil {
			t.Fatalf("accepted %v", value)
		}
	}
	value := 5.0
	for key, rule := range map[string]models.ResourceRule{"filesystem": {MinFreePercent: &value}, "tablespace": {MaxUsedPercent: &value}, "asm_space": {MaxUsedPercent: &value}, "backups": {MaxUsedPercent: &value}, "max_lag": {MinFreePercent: &value}} {
		if ValidateSettings(models.CheckSettings{Resources: map[string]map[string]models.ResourceRule{key: {"resource": rule}}}) == nil {
			t.Fatal("accepted ineffective rule", key)
		}
	}
}

func TestScheduleConfigurationAndDSTBoundaries(t *testing.T) {
	if validateWeekdays([]string{"mon", "mon"}) == nil || validateWeekdays([]string{"monday"}) == nil {
		t.Fatal("invalid weekdays accepted")
	}
	if scheduledDay([]string{}, time.Monday) || !scheduledDay(nil, time.Monday) || !scheduledDay([]string{"mon"}, time.Monday) {
		t.Fatal("weekday semantics")
	}
	if validateScheduleOverlap([]models.Schedule{{Start: "22:00", End: "07:00"}, {Start: "06:00", End: "10:00"}}) == nil {
		t.Fatal("overnight overlap accepted")
	}
	if validateScheduleOverlap([]models.Schedule{{Start: "06:00", End: "10:00"}, {Start: "10:00", End: "14:00"}}) != nil {
		t.Fatal("adjacent half-open windows rejected")
	}
	loc, _ := time.LoadLocation("Europe/London")
	for _, date := range []time.Time{time.Date(2026, 3, 29, 0, 0, 0, 0, loc), time.Date(2026, 10, 25, 0, 0, 0, 0, loc)} {
		if _, _, err := scheduleBounds(date, models.Schedule{Start: "01:30", End: "04:00"}); err == nil {
			t.Fatal("DST ambiguous/nonexistent boundary accepted")
		}
	}
}
