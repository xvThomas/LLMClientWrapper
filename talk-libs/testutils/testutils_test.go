package testutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProjectRoot(t *testing.T) {
	root := GetProjectRoot()
	if root == "" {
		t.Fatal("expected non-empty project root")
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("expected go.mod at %s: %v", root, err)
	}
}

func TestSetupTestEnv(t *testing.T) {
	SetupTestEnv(t)
}

func TestSetupTestEnvWithRequiredVarsOrSkipTest_Skips(t *testing.T) {
	const key = "TESTUTILS_REQUIRED_VAR_UNLIKELY_TO_EXIST_XYZ123"
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Run("skipped", func(t *testing.T) {
		SetupTestEnvWithRequiredVarsOrSkipTest(t, key)
		t.Error("should have been skipped before reaching here")
	})
}

func TestSetupTestEnvWithRequiredVarsOrSkipTest_DoesNotSkip(t *testing.T) {
	const key = "TESTUTILS_REQUIRED_VAR_SET_XYZ123"
	t.Setenv(key, "present")
	SetupTestEnvWithRequiredVarsOrSkipTest(t, key)
}

func TestRequireEnv_Skips(t *testing.T) {
	const key = "TESTUTILS_REQUIRE_ENV_MISSING_XYZ123"
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Run("skipped", func(t *testing.T) {
		got := RequireEnv(t, key)
		if !strings.Contains(got, "") {
			t.Error("should have been skipped")
		}
	})
}

func TestRequireEnv_ReturnsValue(t *testing.T) {
	const key = "TESTUTILS_REQUIRE_ENV_PRESENT_XYZ123"
	t.Setenv(key, "hello")
	got := RequireEnv(t, key)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}
