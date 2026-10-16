package membership

import (
	"sync"
	"sync/atomic"
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
		cachedStatusMu.Unlock()

		deviceCodeMu.Lock()
		cachedDeviceCode = DeviceCodeV7{}
		deviceCodeCached = false
		deviceCodeMu.Unlock()
	})

	cachedStatusMu.Lock()
	cachedStatus = nil
	cachedStatusTime = time.Time{}
	cachedStatusMu.Unlock()

	deviceCodeMu.Lock()
	cachedDeviceCode = DeviceCodeV7{}
	deviceCodeCached = false
	deviceCodeMu.Unlock()
}

func TestStatusFromResponseUsesNewQuotaFields(t *testing.T) {
	status := statusFromResponse(&MemberStatusResponse{
		TierCode:                    "orange_pro",
		TierName:                    "Orange Pro",
		DailyRuntimeMinutes:         180,
		RegularDailyRuntimeMinutes:  60,
		SpecialPeriodRuntimeMinutes: 300,
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
	if status.SpecialPeriodRuntimeMinutes != 300 {
		t.Fatalf("SpecialPeriodRuntimeMinutes = %d, want 300", status.SpecialPeriodRuntimeMinutes)
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
	if status.SpecialPeriodRuntimeMinutes != 120 {
		t.Fatalf("SpecialPeriodRuntimeMinutes = %d, want fallback 120", status.SpecialPeriodRuntimeMinutes)
	}
}

func TestCheckMembershipOfflineDoesNotUseExternalIdentity(t *testing.T) {
	resetMembershipTestGlobals(t)
	appVersion = "1.0.0"
	clientName = "MFAWPF"
	generateDeviceCodeV7 = func() DeviceCodeV7 {
		t.Error("离线模式不应生成设备码")
		return DeviceCodeV7{}
	}
	fetchMemberStatusFn = func(DeviceCodeV7) (*MemberStatusResponse, error) {
		t.Error("离线模式不应联网校验")
		return nil, nil
	}

	status := checkMembership()

	if status.VerificationUnavailable {
		t.Fatal("离线模式不是网络校验失败，不应标记 VerificationUnavailable")
	}
	if status.UpdateRequired {
		t.Fatalf("UpdateRequired = true, want false")
	}
	if !status.IsMember || !status.AllFeaturesUnlocked || !status.UnlimitedRuntime {
		t.Fatal("离线状态未保持 fork 既有行为")
	}
	if status.TierCode != "orange_free" {
		t.Fatalf("TierCode = %q, want orange_free", status.TierCode)
	}
	if status.RegularDailyRuntimeMinutes != 10 {
		t.Fatalf("RegularDailyRuntimeMinutes = %d, want 10", status.RegularDailyRuntimeMinutes)
	}
	if status.DeviceCode != (DeviceCodeV7{}) {
		t.Fatal("离线状态不应包含设备标识")
	}
}

func TestGetMembershipStatusCachesOfflineStatus(t *testing.T) {
	resetMembershipTestGlobals(t)
	appVersion = "1.0.0"
	clientName = "MFAWPF"

	var deviceCodeCalls int
	var fetchCalls int
	generateDeviceCodeV7 = func() DeviceCodeV7 {
		deviceCodeCalls++
		return DeviceCodeV7{CPUHash: "cpu-hash"}
	}
	fetchMemberStatusFn = func(DeviceCodeV7) (*MemberStatusResponse, error) {
		fetchCalls++
		return &MemberStatusResponse{Matched: false, Score: 75}, nil
	}

	first := GetMembershipStatus()
	second := GetMembershipStatus()

	if first != second {
		t.Fatal("GetMembershipStatus() did not return the cached free status")
	}
	if deviceCodeCalls != 0 || fetchCalls != 0 {
		t.Fatalf("离线缓存不应触发外部校验：设备码 %d 次，网络 %d 次", deviceCodeCalls, fetchCalls)
	}
}

func TestGetMembershipStatusRefreshesOfflineStatusAfterExpiry(t *testing.T) {
	resetMembershipTestGlobals(t)
	appVersion = "1.0.0"
	clientName = "MFAWPF"

	var deviceCodeCalls int
	var fetchCalls int
	generateDeviceCodeV7 = func() DeviceCodeV7 {
		deviceCodeCalls++
		return DeviceCodeV7{CPUHash: "cpu-hash"}
	}
	fetchMemberStatusFn = func(DeviceCodeV7) (*MemberStatusResponse, error) {
		fetchCalls++
		return &MemberStatusResponse{Matched: false, Score: 75}, nil
	}

	first := GetMembershipStatus()
	cachedStatusMu.Lock()
	cachedStatusTime = time.Now().Add(-cacheExpiry)
	cachedStatusDuration = cacheExpiry
	cachedStatusMu.Unlock()
	second := GetMembershipStatus()

	if first == second || second.TierCode != "orange_free" || !second.UnlimitedRuntime {
		t.Fatal("缓存过期后应重新生成有效离线状态")
	}
	if deviceCodeCalls != 0 || fetchCalls != 0 {
		t.Fatalf("离线刷新不应触发外部校验：设备码 %d 次，网络 %d 次", deviceCodeCalls, fetchCalls)
	}
}

func TestGetMembershipStatusCoalescesConcurrentOfflineChecks(t *testing.T) {
	resetMembershipTestGlobals(t)
	appVersion = "1.0.0"
	clientName = "MFAWPF"

	var deviceCodeCalls, fetchCalls atomic.Int32
	generateDeviceCodeV7 = func() DeviceCodeV7 {
		deviceCodeCalls.Add(1)
		return DeviceCodeV7{CPUHash: "cpu-hash"}
	}
	fetchMemberStatusFn = func(DeviceCodeV7) (*MemberStatusResponse, error) {
		fetchCalls.Add(1)
		return &MemberStatusResponse{Matched: false, Score: 75}, nil
	}

	const callers = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan *MembershipStatus, callers)
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			<-start
			results <- GetMembershipStatus()
		}()
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("并发离线状态查询超时")
	}
	close(results)
	var first *MembershipStatus
	for status := range results {
		if status == nil {
			t.Fatal("并发查询返回空状态")
		}
		if first == nil {
			first = status
		}
		if status != first {
			t.Fatal("并发查询未共用同一缓存状态")
		}
	}
	if deviceCodeCalls.Load() != 0 || fetchCalls.Load() != 0 {
		t.Fatal("并发离线查询不应生成设备码或发起网络校验")
	}
}
