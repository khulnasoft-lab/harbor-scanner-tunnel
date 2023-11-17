package scan

import (
	"log/slog"
	"time"

	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/etc"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/harbor"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/tunnel"
)

// Clock wraps the Now method. Introduced to allow replacing the global state with fixed clocks to facilitate testing.
// Now returns the current time.
type Clock interface {
	Now() time.Time
}

type SystemClock struct {
}

func (c *SystemClock) Now() time.Time {
	return time.Now()
}

// Transformer wraps the Transform method.
// Transform transforms Tunnel's scan report into Harbor's packages vulnerabilities report.
type Transformer interface {
	Transform(artifact harbor.Artifact, source []tunnel.Vulnerability) harbor.ScanReport
}

type transformer struct {
	clock Clock
}

// NewTransformer constructs a Transformer with the given Clock.
func NewTransformer(clock Clock) Transformer {
	return &transformer{
		clock: clock,
	}
}

func (t *transformer) Transform(artifact harbor.Artifact, source []tunnel.Vulnerability) harbor.ScanReport {
	vulnerabilities := make([]harbor.VulnerabilityItem, len(source))

	for i, v := range source {
		vulnerabilities[i] = harbor.VulnerabilityItem{
			ID:               v.VulnerabilityID,
			Pkg:              v.PkgName,
			Version:          v.InstalledVersion,
			FixVersion:       v.FixedVersion,
			Severity:         t.toHarborSeverity(v.Severity),
			Description:      v.Description,
			Links:            t.toLinks(v.PrimaryURL, v.References),
			Layer:            t.toHarborLayer(v.Layer),
			CweIDs:           v.CweIDs,
			VendorAttributes: t.toVendorAttributes(v.CVSS),
		}
	}

	return harbor.ScanReport{
		GeneratedAt:     t.clock.Now(),
		Scanner:         etc.GetScannerMetadata(),
		Artifact:        artifact,
		Severity:        t.toHighestSeverity(vulnerabilities),
		Vulnerabilities: vulnerabilities,
	}
}

func (t *transformer) toLinks(primaryURL string, references []string) []string {
	if primaryURL != "" {
		return []string{primaryURL}
	}
	if references == nil {
		return []string{}
	}
	return references
}

var tunnelToHarborSeverityMap = map[string]harbor.Severity{
	"CRITICAL": harbor.SevCritical,
	"HIGH":     harbor.SevHigh,
	"MEDIUM":   harbor.SevMedium,
	"LOW":      harbor.SevLow,
	"UNKNOWN":  harbor.SevUnknown,
}

func (t *transformer) toHarborLayer(tLayer *tunnel.Layer) (hLayer *harbor.Layer) {
	if tLayer == nil {
		return
	}
	hLayer = &harbor.Layer{
		Digest: tLayer.Digest,
		DiffID: tLayer.DiffID,
	}
	return
}

func (t *transformer) toHarborSeverity(severity string) harbor.Severity {
	harborSev, ok := tunnelToHarborSeverityMap[severity]
	if !ok {
		slog.Warn("Unknown tunnel severity", slog.String("severity", severity))
		return harbor.SevUnknown
	}

	return harborSev
}

func (t *transformer) toVendorAttributes(info map[string]tunnel.CVSSInfo) map[string]interface{} {
	attributes := make(map[string]interface{})
	if len(info) > 0 {
		attributes["CVSS"] = info
	}
	return attributes
}

func (t *transformer) toHighestSeverity(vlns []harbor.VulnerabilityItem) (highest harbor.Severity) {
	highest = harbor.SevUnknown

	for _, vln := range vlns {
		if vln.Severity > highest {
			highest = vln.Severity

			if highest == harbor.SevCritical {
				break
			}
		}

	}

	return
}
