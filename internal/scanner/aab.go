package scanner

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/The-Crocop/storepreflight/internal/model"
)

const googleDataSafety = "https://support.google.com/googleplay/android-developer/answer/10787469"

func scanAAB(path string, result model.ScanResult) (model.ScanResult, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return result, fmt.Errorf("open AAB: %w", err)
	}
	defer zr.Close()

	if hasPath(zr.File, func(name string) bool { return name == "BundleConfig.pb" }) {
		addPassed(&result)
	} else {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-ANDROID-001", Severity: model.SeverityBlocker,
			Title: "BundleConfig.pb missing",
			Fix: "Provide the final Android App Bundle produced by the Android build toolchain.",
		})
	}

	var manifest []byte
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if name == "base/manifest/AndroidManifest.xml" {
			manifest, err = readZipEntry(f, 8<<20)
			if err != nil {
				return result, fmt.Errorf("read AndroidManifest.xml: %w", err)
			}
			break
		}
	}
	if len(manifest) == 0 {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-ANDROID-002", Severity: model.SeverityBlocker,
			Title: "Base AndroidManifest.xml missing",
			Fix: "Provide an AAB containing base/manifest/AndroidManifest.xml.",
		})
	} else {
		addPassed(&result)
	}

	permissions := []struct{ Value, Label string }{
		{"android.permission.CAMERA", "camera"},
		{"android.permission.RECORD_AUDIO", "microphone"},
		{"android.permission.ACCESS_FINE_LOCATION", "fine-location"},
		{"android.permission.ACCESS_BACKGROUND_LOCATION", "background-location"},
		{"android.permission.READ_CONTACTS", "contacts"},
		{"android.permission.READ_MEDIA_IMAGES", "photos"},
		{"com.google.android.gms.permission.AD_ID", "advertising-id"},
	}
	for _, p := range permissions {
		if containsText(manifest, p.Value) {
			result.Metadata.Permissions = appendUnique(result.Metadata.Permissions, p.Label)
			result.Findings = append(result.Findings, model.Finding{
				ID: "SPF-ANDROID-PERM-" + strings.ToUpper(strings.ReplaceAll(p.Label, "-", "_")),
				Severity: model.SeverityInfo,
				Title: "Privacy-sensitive Android permission detected: " + p.Label,
				Evidence: p.Value,
			})
		}
	}

	sdkHints := []struct{ Needle, Name string }{
		{"firebase-analytics", "Firebase Analytics"},
		{"play-services-ads", "Google Mobile Ads"},
		{"facebook", "Meta/Facebook SDK"},
		{"sentry", "Sentry"},
		{"adjust", "Adjust"},
		{"appsflyer", "AppsFlyer"},
	}
	for _, sdk := range sdkHints {
		if hasPath(zr.File, func(name string) bool { return strings.Contains(strings.ToLower(name), sdk.Needle) }) {
			result.Metadata.SDKs = appendUnique(result.Metadata.SDKs, sdk.Name)
		}
	}
	if len(result.Metadata.Permissions) > 0 || len(result.Metadata.SDKs) > 0 {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-ANDROID-020", Severity: model.SeverityWarning,
			Title: "Review Google Play Data safety answers for this release",
			Evidence: "Sensitive permissions or SDK signals were detected.",
			Fix: "Confirm the Play Console Data safety declaration matches the data practices of this exact build.",
			Source: googleDataSafety,
		})
	}
	result.Findings = append(result.Findings, model.Finding{
		ID: "SPF-ANDROID-030", Severity: model.SeverityWarning,
		Title: "targetSdk policy validation requires decoded manifest metadata",
		Fix: "StorePreflight Cloud will correlate targetSdk and current Play policy deadlines; full binary decoding is planned for the next scanner milestone.",
	})
	result.Recount()
	return result, nil
}
