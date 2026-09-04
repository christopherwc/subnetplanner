// Command subnetplanner-gui launches the Subnet Planner desktop application.
package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/christopherwc/subnetplanner/internal/guiapp"
)

func main() {
	a := guiapp.NewApp(app.New())
	a.Run()
}
