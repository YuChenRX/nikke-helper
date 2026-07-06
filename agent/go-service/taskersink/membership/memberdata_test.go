package membership

import (
	"errors"
	"testing"
	"time"
)

func resetMembershipTestGlobals(t *testing.T) {
	t.Helper()

	oldAppVersion := appVersion
	oldClientName := clientName
	oldGenerateDeviceCodeV7 := generateDeviceCodeV7
	oldFetchMemberStatusFn := fetchMemberStatusFn

	t.Cleanup(func() {
		appVersion = oldAppVersion
		clientName = oldClientName
		generateDeviceCodeV7 = oldGenerateDeviceCodeV7
		fetchMemberStatusFn = oldFetchMemberStatusFn

		cachedStatusMu.Lock()
		cachedStatus = nil
		cachedStatusTime = time.Time{}
		cachedDeviceCode = DeviceCodeV7{}
		cachedStatusMu.Unlock()
	})

	cachedStatusMu.Lock()
	cachedStatus = nil
	cachedStatusTime = time.Time{}
	cachedDeviceCode = DeviceCodeV7{}
	cachedStatusMu.Unlock()
}

func TestStatusFromResponseUsesNewQuotaFields(t *testing.T) {
	status := statusFromResponse(&MemberStatusResponse{
		TierCode:                    "orange_pro",
		TierName:                    "Orange Pro",
		DailyRuntimeMinutes:         180,
		RegularDailyRuntimeMinutes:  60,
		SpecialPeriodRuntimeMinutes: 1500,
		PaidThroughOn:               "20260701",
		HasFutureRenewal:            true,
	}, DeviceCodeV7{})

	if status.PaidThroughOn != "20260701" {
		t.Fatalf("PaidThroughOn = %q, want 20260701", status.PaidThroughOn)
	}
	if !status.HasFutureRenewal {
		t.Fatalf("HasFutureRenewal = false, want true")
	}
	if status.RegularDailyRuntimeMinutes != 60 {
		t.Fatalf("RegularDailyRuntimeMinutes = %d, want 60", status.RegularDailyRuntimeMinutes)
	}
	if status.DailyRuntimeMinutes != 60 {
		t.Fatalf("DailyRuntimeMinutes = %d, want compatibility alias 60", status.DailyRuntimeMinutes)
	}
	if status.SpecialPeriodRuntimeMinutes != 1500 {
		t.Fatalf("SpecialPeriodRuntimeMinutes = %d, want 1500", status.SpecialPeriodRuntimeMinutes)
	}
}

func TestStatusFromResponseFallsBackToTierSpecialQuota(t *testing.T) {
	status := statusFromResponse(&MemberStatusResponse{
		TierCode:            "orange_plus",
		TierName:            "Orange Plus",
		DailyRuntimeMinutes: 30,
	}, DeviceCodeV7{})

	if status.RegularDailyRuntimeMinutes != 30 {
		t.Fatalf("RegularDailyRuntimeMinutes = %d, want 30", status.RegularDailyRuntimeMinutes)
	}
	if status.SpecialPeriodRuntimeMinutes != 600 {
		t.Fatalf("SpecialPeriodRuntimeMinutes = %d, want fallback 600", status.SpecialPeriodRuntimeMinutes)
	}
}

func TestCheckMembershipUnavailableFallsBackToFreeStatus(t *testing.T) {
	resetMembershipTestGlobals(t)
	appVersion = "1.0.0"
	clientName = "MFAWPF"
	generateDeviceCodeV7 = func() DeviceCodeV7 {
		return DeviceCodeV7{CPUHash: "cpu-hash"}
	}
	fetchMemberStatusFn = func(DeviceCodeV7) (*MemberStatusResponse, error) {
		return nil, errors.New("temporary service failure")
	}

	status := checkMembership()

	if !status.VerificationUnavailable {
		t.Fatalf("VerificationUnavailable = false, want true")
	}
	if status.UpdateRequired {
		t.Fatalf("UpdateRequired = true, want false")
	}
	if status.IsMember {
		t.Fatalf("IsMember = true, want false")
	}
	if status.TierCode != "orange_free" {
		t.Fatalf("TierCode = %q, want orange_free", status.TierCode)
	}
	if status.RegularDailyRuntimeMinutes != 10 {
		t.Fatalf("RegularDailyRuntimeMinutes = %d, want 10", status.RegularDailyRuntimeMinutes)
	}
	if status.DeviceCode.CPUHash != "cpu-hash" {
		t.Fatalf("DeviceCode.CPUHash = %q, want cpu-hash", status.DeviceCode.CPUHash)
	}
}
