// Package guiapp builds the Subnet Planner Fyne user interface on top of
// the internal/subnet planning engine. It is kept separate from cmd/ so the
// widget-wiring logic can be exercised by tests using Fyne's headless test
// driver, without needing a real display.
package guiapp

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/christopherwc/subnetplanner/internal/subnet"
)

// App wraps the Fyne application and window along with the widgets that
// tests need to drive and inspect.
type App struct {
	FyneApp fyne.App
	Window  fyne.Window

	// Subnet Info tab
	infoCIDREntry  *widget.Entry
	infoResult     *widget.Label
	infoErrorLabel *widget.Label

	// Equal Split tab
	splitCIDREntry  *widget.Entry
	splitModeSelect *widget.Select // "By count" or "By new prefix"
	splitValueEntry *widget.Entry
	splitResult     *widget.Label
	splitErrorLabel *widget.Label

	// VLSM Planner tab
	vlsmCIDREntry  *widget.Entry
	vlsmReqsEntry  *widget.Entry
	vlsmResult     *widget.Label
	vlsmErrorLabel *widget.Label
}

// NewApp constructs the application and its window but does not show or run
// it, so callers (including tests) control the lifecycle.
func NewApp(fa fyne.App) *App {
	a := &App{FyneApp: fa}
	a.Window = fa.NewWindow("Subnet Planner")
	a.Window.SetContent(a.buildUI())
	a.Window.Resize(fyne.NewSize(760, 560))
	return a
}

// Run shows the window and starts the Fyne event loop. Not exercised by
// unit tests (it blocks), which instead drive the widgets directly.
func (a *App) Run() {
	a.Window.ShowAndRun()
}

func (a *App) buildUI() fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Subnet Info", a.buildInfoTab()),
		container.NewTabItem("Equal Split", a.buildSplitTab()),
		container.NewTabItem("VLSM Planner", a.buildVLSMTab()),
	)
	return tabs
}

// ---- Subnet Info tab ----

func (a *App) buildInfoTab() fyne.CanvasObject {
	a.infoCIDREntry = widget.NewEntry()
	a.infoCIDREntry.SetPlaceHolder("e.g. 192.168.1.0/24 or 2001:db8::/64")

	a.infoResult = widget.NewLabel("")
	a.infoResult.Wrapping = fyne.TextWrapWord
	a.infoErrorLabel = widget.NewLabel("")
	a.infoErrorLabel.Importance = widget.DangerImportance

	button := widget.NewButton("Get Details", a.onGetDetails)
	a.infoCIDREntry.OnSubmitted = func(string) { a.onGetDetails() }

	form := widget.NewForm(widget.NewFormItem("CIDR", a.infoCIDREntry))

	return container.NewVBox(
		widget.NewLabel("Enter a network in CIDR notation to see its details."),
		form,
		button,
		a.infoErrorLabel,
		a.infoResult,
	)
}

func (a *App) onGetDetails() {
	cidr := strings.TrimSpace(a.infoCIDREntry.Text)
	d, err := subnet.GetDetails(cidr)
	if err != nil {
		a.infoErrorLabel.SetText(err.Error())
		a.infoResult.SetText("")
		return
	}
	a.infoErrorLabel.SetText("")
	a.infoResult.SetText(formatDetails(d))
}

func formatDetails(d subnet.Details) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Network:        %s\n", d.CIDR)
	fmt.Fprintf(&b, "Address family: %s\n", family(d.IsIPv6))
	fmt.Fprintf(&b, "First usable:   %s\n", d.FirstUsable)
	fmt.Fprintf(&b, "Last usable:    %s\n", d.LastUsable)
	if !d.IsIPv6 {
		fmt.Fprintf(&b, "Broadcast:      %s\n", d.Broadcast)
	}
	fmt.Fprintf(&b, "Total addresses: %s\n", d.TotalAddresses.String())
	fmt.Fprintf(&b, "Usable hosts:    %s\n", d.UsableHosts.String())
	return b.String()
}

func family(isIPv6 bool) string {
	if isIPv6 {
		return "IPv6"
	}
	return "IPv4"
}

// ---- Equal Split tab ----

func (a *App) buildSplitTab() fyne.CanvasObject {
	a.splitCIDREntry = widget.NewEntry()
	a.splitCIDREntry.SetPlaceHolder("e.g. 10.0.0.0/24")

	a.splitModeSelect = widget.NewSelect([]string{"By subnet count", "By new prefix length"}, nil)
	a.splitModeSelect.SetSelected("By subnet count")

	a.splitValueEntry = widget.NewEntry()
	a.splitValueEntry.SetPlaceHolder("e.g. 4")

	a.splitResult = widget.NewLabel("")
	a.splitResult.Wrapping = fyne.TextWrapWord
	a.splitErrorLabel = widget.NewLabel("")
	a.splitErrorLabel.Importance = widget.DangerImportance

	button := widget.NewButton("Split", a.onSplit)

	form := widget.NewForm(
		widget.NewFormItem("Base CIDR", a.splitCIDREntry),
		widget.NewFormItem("Mode", a.splitModeSelect),
		widget.NewFormItem("Value", a.splitValueEntry),
	)

	return container.NewVBox(
		widget.NewLabel("Split a base network into equal-sized subnets."),
		form,
		button,
		a.splitErrorLabel,
		container.NewVScroll(a.splitResult),
	)
}

func (a *App) onSplit() {
	cidr := strings.TrimSpace(a.splitCIDREntry.Text)
	valStr := strings.TrimSpace(a.splitValueEntry.Text)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		a.splitErrorLabel.SetText("value must be a whole number")
		a.splitResult.SetText("")
		return
	}

	var subs []interface{ String() string }
	var splitErr error

	if a.splitModeSelect.Selected == "By new prefix length" {
		prefixes, e := subnet.SplitByPrefix(cidr, val)
		splitErr = e
		for _, p := range prefixes {
			subs = append(subs, p)
		}
	} else {
		prefixes, e := subnet.SplitCount(cidr, val)
		splitErr = e
		for _, p := range prefixes {
			subs = append(subs, p)
		}
	}

	if splitErr != nil {
		a.splitErrorLabel.SetText(splitErr.Error())
		a.splitResult.SetText("")
		return
	}
	a.splitErrorLabel.SetText("")

	var b strings.Builder
	fmt.Fprintf(&b, "%d subnet(s):\n", len(subs))
	for _, s := range subs {
		fmt.Fprintf(&b, "  %s\n", s.String())
	}
	a.splitResult.SetText(b.String())
}

// ---- VLSM Planner tab ----

func (a *App) buildVLSMTab() fyne.CanvasObject {
	a.vlsmCIDREntry = widget.NewEntry()
	a.vlsmCIDREntry.SetPlaceHolder("e.g. 192.168.1.0/24")

	a.vlsmReqsEntry = widget.NewMultiLineEntry()
	a.vlsmReqsEntry.SetPlaceHolder("One requirement per line: Name,Hosts\nSales,50\nEngineering,100\nGuest,10")
	a.vlsmReqsEntry.SetMinRowsVisible(6)

	a.vlsmResult = widget.NewLabel("")
	a.vlsmResult.Wrapping = fyne.TextWrapWord
	a.vlsmErrorLabel = widget.NewLabel("")
	a.vlsmErrorLabel.Importance = widget.DangerImportance

	button := widget.NewButton("Plan", a.onPlanVLSM)

	form := widget.NewForm(widget.NewFormItem("Base CIDR", a.vlsmCIDREntry))

	return container.NewVBox(
		widget.NewLabel("Allocate variable-length subnets (VLSM) from a base network."),
		form,
		widget.NewLabel("Requirements (one per line as Name,Hosts):"),
		a.vlsmReqsEntry,
		button,
		a.vlsmErrorLabel,
		container.NewVScroll(a.vlsmResult),
	)
}

// ParseRequirements parses the "Name,Hosts" textarea format into
// subnet.Requirement values. Exported for direct unit testing.
func ParseRequirements(text string) ([]subnet.Requirement, error) {
	var reqs []subnet.Requirement
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: expected format Name,Hosts", i+1)
		}
		name := strings.TrimSpace(parts[0])
		hostsStr := strings.TrimSpace(parts[1])
		hosts, err := strconv.Atoi(hostsStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: %q is not a whole number", i+1, hostsStr)
		}
		if name == "" {
			return nil, fmt.Errorf("line %d: name must not be empty", i+1)
		}
		reqs = append(reqs, subnet.Requirement{Name: name, Hosts: hosts})
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("enter at least one requirement")
	}
	return reqs, nil
}

func (a *App) onPlanVLSM() {
	cidr := strings.TrimSpace(a.vlsmCIDREntry.Text)
	reqs, err := ParseRequirements(a.vlsmReqsEntry.Text)
	if err != nil {
		a.vlsmErrorLabel.SetText(err.Error())
		a.vlsmResult.SetText("")
		return
	}

	allocs, err := subnet.PlanVLSM(cidr, reqs)
	if err != nil {
		a.vlsmErrorLabel.SetText(err.Error())
		a.vlsmResult.SetText("")
		return
	}
	a.vlsmErrorLabel.SetText("")

	var b strings.Builder
	for _, alloc := range allocs {
		fmt.Fprintf(&b, "%-15s %-18s usable hosts: %s (requested %d)\n",
			alloc.Name, alloc.CIDR, alloc.UsableHosts.String(), alloc.HostsRequested)
	}
	a.vlsmResult.SetText(b.String())
}
