package guiapp

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func newTestApp() *App {
	return NewApp(test.NewApp())
}

func TestRun(t *testing.T) {
	// The headless test driver's Run() is a no-op, so this simply exercises
	// the Run method wiring without blocking.
	a := newTestApp()
	a.Run()
}

func TestNewApp(t *testing.T) {
	a := newTestApp()
	if a.Window == nil {
		t.Fatal("Window is nil")
	}
	if a.Window.Title() != "Subnet Planner" {
		t.Errorf("Title = %q, want %q", a.Window.Title(), "Subnet Planner")
	}
	if a.Window.Content() == nil {
		t.Fatal("Content is nil")
	}
}

func TestOnGetDetailsValid(t *testing.T) {
	a := newTestApp()
	a.infoCIDREntry.SetText("192.168.1.0/24")
	a.onGetDetails()

	if a.infoErrorLabel.Text != "" {
		t.Errorf("unexpected error: %q", a.infoErrorLabel.Text)
	}
	if !strings.Contains(a.infoResult.Text, "192.168.1.0/24") {
		t.Errorf("result missing CIDR: %q", a.infoResult.Text)
	}
	if !strings.Contains(a.infoResult.Text, "Broadcast:") {
		t.Errorf("result missing broadcast for IPv4: %q", a.infoResult.Text)
	}
}

func TestOnGetDetailsIPv6(t *testing.T) {
	a := newTestApp()
	a.infoCIDREntry.SetText("  2001:db8::/64  ")
	a.onGetDetails()

	if a.infoErrorLabel.Text != "" {
		t.Errorf("unexpected error: %q", a.infoErrorLabel.Text)
	}
	if strings.Contains(a.infoResult.Text, "Broadcast:") {
		t.Errorf("result should not contain broadcast for IPv6: %q", a.infoResult.Text)
	}
	if !strings.Contains(a.infoResult.Text, "IPv6") {
		t.Errorf("result missing address family: %q", a.infoResult.Text)
	}
}

func TestOnGetDetailsInvalid(t *testing.T) {
	a := newTestApp()
	a.infoCIDREntry.SetText("not-a-cidr")
	a.onGetDetails()

	if a.infoErrorLabel.Text == "" {
		t.Error("expected error message")
	}
	if a.infoResult.Text != "" {
		t.Errorf("expected empty result on error, got %q", a.infoResult.Text)
	}

	// Submitting via the entry's OnSubmitted callback should behave the same.
	a.infoCIDREntry.SetText("192.168.1.0/24")
	a.infoCIDREntry.OnSubmitted("192.168.1.0/24")
	if a.infoErrorLabel.Text != "" {
		t.Errorf("unexpected error after submit: %q", a.infoErrorLabel.Text)
	}
}

func TestOnSplitByCount(t *testing.T) {
	a := newTestApp()
	a.splitCIDREntry.SetText("10.0.0.0/24")
	a.splitModeSelect.SetSelected("By subnet count")
	a.splitValueEntry.SetText("4")
	a.onSplit()

	if a.splitErrorLabel.Text != "" {
		t.Fatalf("unexpected error: %q", a.splitErrorLabel.Text)
	}
	if !strings.Contains(a.splitResult.Text, "4 subnet(s)") {
		t.Errorf("result missing count: %q", a.splitResult.Text)
	}
	if !strings.Contains(a.splitResult.Text, "10.0.0.0/26") {
		t.Errorf("result missing first subnet: %q", a.splitResult.Text)
	}
}

func TestOnSplitByPrefix(t *testing.T) {
	a := newTestApp()
	a.splitCIDREntry.SetText("10.0.0.0/24")
	a.splitModeSelect.SetSelected("By new prefix length")
	a.splitValueEntry.SetText("26")
	a.onSplit()

	if a.splitErrorLabel.Text != "" {
		t.Fatalf("unexpected error: %q", a.splitErrorLabel.Text)
	}
	if !strings.Contains(a.splitResult.Text, "10.0.0.192/26") {
		t.Errorf("result missing last subnet: %q", a.splitResult.Text)
	}
}

func TestOnSplitInvalidValue(t *testing.T) {
	a := newTestApp()
	a.splitCIDREntry.SetText("10.0.0.0/24")
	a.splitModeSelect.SetSelected("By subnet count")
	a.splitValueEntry.SetText("not-a-number")
	a.onSplit()

	if a.splitErrorLabel.Text == "" {
		t.Error("expected error for non-numeric value")
	}
	if a.splitResult.Text != "" {
		t.Errorf("expected empty result on error, got %q", a.splitResult.Text)
	}
}

func TestOnSplitInvalidCIDR(t *testing.T) {
	a := newTestApp()
	a.splitCIDREntry.SetText("not-a-cidr")
	a.splitModeSelect.SetSelected("By subnet count")
	a.splitValueEntry.SetText("2")
	a.onSplit()

	if a.splitErrorLabel.Text == "" {
		t.Error("expected error for invalid CIDR")
	}
}

func TestParseRequirements(t *testing.T) {
	reqs, err := ParseRequirements("Sales,50\nEngineering,100\n\n  Guest , 10 \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 3 {
		t.Fatalf("got %d requirements, want 3", len(reqs))
	}
	if reqs[0].Name != "Sales" || reqs[0].Hosts != 50 {
		t.Errorf("reqs[0] = %+v", reqs[0])
	}
	if reqs[2].Name != "Guest" || reqs[2].Hosts != 10 {
		t.Errorf("reqs[2] = %+v", reqs[2])
	}
}

func TestParseRequirementsErrors(t *testing.T) {
	cases := []string{
		"",
		"   \n   ",
		"NoComma",
		"Name,not-a-number",
		",50",
		"  ,50",
	}
	for _, c := range cases {
		if _, err := ParseRequirements(c); err == nil {
			t.Errorf("ParseRequirements(%q) expected error, got nil", c)
		}
	}
}

func TestOnPlanVLSMValid(t *testing.T) {
	a := newTestApp()
	a.vlsmCIDREntry.SetText("192.168.1.0/24")
	a.vlsmReqsEntry.SetText("Sales,50\nEngineering,100\nGuest,10")
	a.onPlanVLSM()

	if a.vlsmErrorLabel.Text != "" {
		t.Fatalf("unexpected error: %q", a.vlsmErrorLabel.Text)
	}
	for _, want := range []string{"Sales", "Engineering", "Guest"} {
		if !strings.Contains(a.vlsmResult.Text, want) {
			t.Errorf("result missing %q: %q", want, a.vlsmResult.Text)
		}
	}
}

func TestOnPlanVLSMBadRequirements(t *testing.T) {
	a := newTestApp()
	a.vlsmCIDREntry.SetText("192.168.1.0/24")
	a.vlsmReqsEntry.SetText("bad-line-no-comma")
	a.onPlanVLSM()

	if a.vlsmErrorLabel.Text == "" {
		t.Error("expected error for malformed requirements")
	}
	if a.vlsmResult.Text != "" {
		t.Errorf("expected empty result on error, got %q", a.vlsmResult.Text)
	}
}

func TestOnPlanVLSMOutOfSpace(t *testing.T) {
	a := newTestApp()
	a.vlsmCIDREntry.SetText("192.168.1.0/30")
	a.vlsmReqsEntry.SetText("TooBig,1000")
	a.onPlanVLSM()

	if a.vlsmErrorLabel.Text == "" {
		t.Error("expected error when requirement exceeds capacity")
	}
}
