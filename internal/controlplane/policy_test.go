package controlplane

import (
	"testing"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

func TestValidatePolicyDailyMinutes(t *testing.T) {
	t.Parallel()
	err := ValidatePolicy(ChildPolicy{DailyMinutes: -1, LocalOnly: true})
	if err == nil {
		t.Fatal("expected error for negative daily_minutes")
	}
	var ae *apperr.Error
	if !asAppErr(err, &ae) || ae.Code != apperr.CodeInvalid {
		t.Fatalf("got %#v", err)
	}

	err = ValidatePolicy(ChildPolicy{DailyMinutes: MaxDailyMinutes + 1, LocalOnly: true})
	if err == nil {
		t.Fatal("expected error for oversized daily_minutes")
	}

	if err := ValidatePolicy(DefaultChildPolicy()); err != nil {
		t.Fatalf("default policy: %v", err)
	}
}

func TestValidatePolicyBedtime(t *testing.T) {
	t.Parallel()
	p := DefaultChildPolicy()
	p.BedtimeStart = "21:00"
	if err := ValidatePolicy(p); err == nil {
		t.Fatal("expected error when only bedtime_start is set")
	}
	p.BedtimeEnd = "07:00"
	if err := ValidatePolicy(p); err != nil {
		t.Fatalf("valid bedtime: %v", err)
	}
	p.BedtimeStart = "9pm"
	if err := ValidatePolicy(p); err == nil {
		t.Fatal("expected error for invalid bedtime format")
	}
}

func TestValidateCreateName(t *testing.T) {
	t.Parallel()
	if err := ValidateCreateName("  "); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := ValidateCreateName("Ada"); err != nil {
		t.Fatalf("valid name: %v", err)
	}
}

func TestDefaultChildPolicyFailClosed(t *testing.T) {
	t.Parallel()
	p := DefaultChildPolicy()
	if !p.LocalOnly || !p.HideShorts || !p.HideLive {
		t.Fatalf("expected fail-closed defaults, got %+v", p)
	}
}

func asAppErr(err error, target **apperr.Error) bool {
	if err == nil {
		return false
	}
	ae, ok := err.(*apperr.Error)
	if !ok {
		return false
	}
	*target = ae
	return true
}
