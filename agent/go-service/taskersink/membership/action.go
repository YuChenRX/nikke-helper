package membership

import (
	"fmt"

	"github.com/1204244136/MDA/agent/go-service/pkg/i18n"
	"github.com/1204244136/MDA/agent/go-service/pkg/maafocus"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// MembershipCheckAction is a custom action that checks membership before executing member-only tasks.
// It runs synchronously in the pipeline, blocking execution for non-members.
type MembershipCheckAction struct{}

var _ maa.CustomActionRunner = &MembershipCheckAction{}

func (a *MembershipCheckAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	status := GetMembershipStatus()

	if status.UnsupportedTier {
		log.Warn().
			Str("tier", status.MembershipType).
			Msg("MembershipCheck: unsupported tier")
	}

	if status.IsMember {
		log.Info().
			Str("tier", status.MembershipType).
			Int("level", status.UserLevel).
			Str("expiry", status.VirtualExpiry).
			Msg("MembershipCheck: member verified, allowing")
		maafocus.Print(ctx, fmt.Sprintf(
			i18n.T("tasker.membership_check.verified"),
			status.MembershipType, status.VirtualExpiry,
		))
		return true
	}

	sponsorURL := fmt.Sprintf(
		"https://doropay.top?cpu=%s&uuid=%s&bios=%s&board=%s&disk=%s&guid=%s",
		status.DeviceCode.CPUHash,
		status.DeviceCode.UUIDHash,
		status.DeviceCode.BIOSHash,
		status.DeviceCode.BoardHash,
		status.DeviceCode.DiskHash,
		status.DeviceCode.GUIDHash,
	)
	maafocus.Print(ctx, fmt.Sprintf(
		i18n.T("tasker.membership_check.denied"),
		sponsorURL,
	))

	return false
}
