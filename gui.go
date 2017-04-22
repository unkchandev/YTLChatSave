package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var mw = &LogWindow{}

type LogWindow struct {
	*walk.MainWindow

	logTE *walk.TextEdit
}

func runMainWindow() {
	MW := MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "YouTube Live Chat Save",
		MinSize:  Size{640, 480},
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{AssignTo: &mw.logTE, ReadOnly: true},
		},
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Separator{},
					Action{
						Text: "Preference",
						OnTriggered: func() {
							if cmd, err := runConfigDialog(mw); err != nil {
								logch <- err.Error()
							} else if cmd == walk.DlgCmdOK {
								logch <- "Config file changed. Please restart."
							}
						},
					},
					Action{
						Text:        "Exit",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
		},
	}
	MW.Run()
}

func runConfigDialog(owner walk.Form) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	yc, err := loadConfig()
	if err != nil {
		return walk.DlgCmdAbort, err
	}

	return Dialog{
		AssignTo:      &dlg,
		Title:         "Preference",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{450, 200},
		Layout:        VBox{},
		DataBinder: DataBinder{
			AssignTo:   &db,
			DataSource: yc,
		},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: "APIKey:",
					},
					LineEdit{
						Text: Bind("APIKey"),
					},
					Label{
						Text: "ChannelID:",
					},
					LineEdit{
						Text: Bind("ChannelID"),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "OK",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								logch <- "Unable to save config file. Error: " + err.Error()
								return
							}
							saveConfigFile(yc)
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			}},
	}.Run(owner)
}
