package scanner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestScanIPA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "App.ipa")
	writeZip(t, path, map[string]string{
		"Payload/Demo.app/Info.plist": "<plist><key>NSCameraUsageDescription</key><string>Scan codes</string></plist>",
		"Payload/Demo.app/PrivacyInfo.xcprivacy": "{}",
	})
	result, err := Scan(path)
	if err != nil { t.Fatal(err) }
	if result.Platform != "ios" { t.Fatalf("platform = %q", result.Platform) }
	if result.Summary.Blockers != 0 { t.Fatalf("unexpected blockers: %#v", result.Findings) }
	if !contains(result.Metadata.Permissions, "camera") { t.Fatalf("camera permission not detected: %#v", result.Metadata.Permissions) }
}

func TestScanAAB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	writeZip(t, path, map[string]string{
		"BundleConfig.pb": "demo",
		"base/manifest/AndroidManifest.xml": "android.permission.CAMERA",
	})
	result, err := Scan(path)
	if err != nil { t.Fatal(err) }
	if result.Platform != "android" { t.Fatalf("platform = %q", result.Platform) }
	if result.Summary.Blockers != 0 { t.Fatalf("unexpected blockers: %#v", result.Findings) }
	if !contains(result.Metadata.Permissions, "camera") { t.Fatalf("camera permission not detected: %#v", result.Metadata.Permissions) }
}

func TestUnsupportedArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.zip")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil { t.Fatal(err) }
	result, err := Scan(path)
	if err != nil { t.Fatal(err) }
	if result.Summary.Blockers != 1 { t.Fatalf("expected one blocker, got %#v", result.Summary) }
}

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil { t.Fatal(err) }
	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil { t.Fatal(err) }
		if _, err := w.Write([]byte(content)); err != nil { t.Fatal(err) }
	}
	if err := zw.Close(); err != nil { t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }
}
func contains(values []string, needle string) bool {
	for _, value := range values { if value == needle { return true } }
	return false
}
