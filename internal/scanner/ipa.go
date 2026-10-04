package scanner

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/The-Crocop/storeready/internal/model"
)

const applePrivacyManifest = "https://developer.apple.com/documentation/bundleresources/privacy_manifest_files"

func scanIPA(path string, result model.ScanResult) (model.ScanResult, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return result, fmt.Errorf("open IPA: %w", err)
	}
	defer zr.Close()

	var infoPlist []byte
	var infoPath string
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.HasPrefix(name, "Payload/") && strings.Contains(name, ".app/") && strings.HasSuffix(name, "/Info.plist") {
			infoPlist, err = readZipEntry(f, 4<<20)
			if err != nil {
				return result, fmt.Errorf("read Info.plist: %w", err)
			}
			infoPath = name
			break
		}
	}
	if len(infoPlist) == 0 {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SR-IOS-001", Severity: model.SeverityBlocker,
			Title: "Info.plist was not found in the application bundle",
			Fix: "Scan the final exported IPA containing Payload/<App>.app/Info.plist.",
		})
	} else {
		addPassed(&result)
		result.Findings = append(result.Findings, model.Finding{
			ID: "SR-IOS-INFO-001", Severity: model.SeverityInfo,
			Title: "Application Info.plist found", Evidence: infoPath,
		})
	}

	if hasPath(zr.File, func(name string) bool { return strings.HasSuffix(name, "/PrivacyInfo.xcprivacy") }) {
		addPassed(&result)
	} else {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SR-IOS-002", Severity: model.SeverityWarning,
			Title: "No PrivacyInfo.xcprivacy found in the IPA",
			Fix: "Verify whether the app or included SDKs use APIs that require a privacy manifest.",
			Source: applePrivacyManifest,
		})
	}
	if hasPath(zr.File, func(name string) bool { return strings.HasSuffix(name, "/embedded.mobileprovision") }) {
		addPassed(&result)
	}

	permissionKeys := []struct{ Key, Label string }{
		{"NSCameraUsageDescription", "camera"},
		{"NSMicrophoneUsageDescription", "microphone"},
		{"NSLocationWhenInUseUsageDescription", "location-when-in-use"},
		{"NSLocationAlwaysAndWhenInUseUsageDescription", "location-always"},
		{"NSPhotoLibraryUsageDescription", "photo-library"},
		{"NSContactsUsageDescription", "contacts"},
	}
	for _, p := range permissionKeys {
		if containsText(infoPlist, p.Key) {
			result.Metadata.Permissions = appendUnique(result.Metadata.Permissions, p.Label)
			result.Findings = append(result.Findings, model.Finding{
				ID: "SR-IOS-PERM-" + strings.ToUpper(strings.ReplaceAll(p.Label, "-", "_")),
				Severity: model.SeverityInfo,
				Title: "Privacy-sensitive capability declared: " + p.Label,
				Evidence: p.Key,
			})
		}
	}

	sdkHints := []struct{ Needle, Name string }{
		{"FirebaseAnalytics", "Firebase Analytics"},
		{"GoogleAppMeasurement", "Google App Measurement"},
		{"FBSDK", "Meta/Facebook SDK"},
		{"AppTrackingTransparency", "AppTrackingTransparency"},
		{"Sentry", "Sentry"},
	}
	for _, sdk := range sdkHints {
		if hasPath(zr.File, func(name string) bool { return strings.Contains(strings.ToLower(name), strings.ToLower(sdk.Needle)) }) {
			result.Metadata.SDKs = appendUnique(result.Metadata.SDKs, sdk.Name)
		}
	}
	if len(result.Metadata.SDKs) > 0 {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SR-IOS-020", Severity: model.SeverityWarning,
			Title: "Third-party SDKs with privacy implications detected",
			Evidence: strings.Join(result.Metadata.SDKs, ", "),
			Fix: "Confirm App Store privacy answers and required SDK privacy manifests are still accurate for this release.",
		})
	}
	result.Recount()
	return result, nil
}
