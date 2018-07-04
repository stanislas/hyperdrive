package main

import (
	"github.com/DEEP-IMPACT-AG/hyperdrive/hview"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"log"
)

func main2() {
	app := tview.NewApplication()
	menu := hview.NewDropDown()
	table := tview.NewTable()
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(populateMenu(menu), 1, 0, true).
		AddItem(table.SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(menu)
		}), 0, 1, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		log.Fatal(err)
	}
}

func populateMenu(menu *hview.SearchBar) *hview.SearchBar {
	return menu.
		AddOption("tag:", nil).
		AddOption("action!:", nil).
		AddOption("resource:", nil)
}

func main() {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	app := tview.NewApplication()
	pages := tview.NewPages()
	menu := tview.NewList()
	submenu := tview.NewList().SetDoneFunc(func() {
		app.SetFocus(menu)
	})
	menu.
		AddItem("Browse", "Browse AWS", 'b', func() {
			submenu.Clear()
			submenu.AddItem("Cloudformation", "", 'c',
				func() {
					if !pages.HasPage("browser") {
						table := fetchCFS(cfg, app)
						pages.AddPage("browser", table, true, true)
					}
					pages.SwitchToPage("browser")
					app.SetFocus(pages)
				})
			app.SetFocus(submenu)
		}).
		AddItem("Create", "Create Resources", 'c', func() {
			submenu.Clear()
			submenu.AddItem("DefaultVPC", "", 'v', nil)
			submenu.AddItem("HostedZone", "", 'z', nil)
			app.SetFocus(submenu)
		}).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})
	menus := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(menu, 0, 1, true).
		AddItem(submenu, 0, 1, true)
	flex := tview.NewFlex().
		AddItem(menus, 30, 1, true).
		AddItem(pages, 0, 1, false)
	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}

func fetchCFS(cfg aws.Config, app *tview.Application) tview.Primitive {
	table := tview.NewTable()
	table.
		SetCell(0, 0, headerCell("Stack Name")).
		SetCell(0, 1, headerCell("Created Time")).
		SetCell(0, 2, headerCell("Status")).
		SetCell(0, 3, headerCell("Description")).
		SetFixed(1, 0).
		SetSelectable(true, true)
	go func() {
		cfs := cloudformation.New(cfg)
		request := cloudformation.DescribeStacksInput{}
		res, err := cfs.DescribeStacksRequest(&request).Send()
		if err != nil {
			panic(err)
		}
		for i, stack := range res.Stacks {
			row := i + 1
			table.
				SetCell(row, 0, tview.NewTableCell(*stack.StackName)).
				SetCell(row, 1, tview.NewTableCell(stack.CreationTime.String())).
				SetCell(row, 2, tview.NewTableCell(string(stack.StackStatus)))
		}
		app.Draw()
	}()
	return table
}

func headerCell(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetAlign(tview.AlignLeft)
}
