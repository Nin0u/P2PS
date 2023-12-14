package main

import (
	"fmt"
	"image/color"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func renderListScreen(a fyne.App, w fyne.Window, list []string) {
	current_selected := -1
	work := true

	element := widget.NewList(
		// func that returns the number of items in the list
		func() int {
			return len(list)
		},
		// func that returns the component structure of the List Item
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		// func that is called for each item in the list and allows
		// you to show the content on the previously defined ui structure
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(list[i])
		})

	element.OnSelected = func(id widget.ListItemID) {
		current_selected = id
	}

	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = color.Transparent

	content := container.NewBorder(
		nil, // TOP of the container

		// this will be a the BOTTOM of the container
		container.NewGridWithColumns(
			3,

			widget.NewButton("Export", func() {
				if !work {
					return
				}
				other_window := a.NewWindow("File Explorer")
				other_window.Resize(fyne.NewSize(600, 400))
				other_window.Show()

				file_Dialog := dialog.NewFileOpen(
					func(r fyne.URIReadCloser, _ error) {

						fmt.Println(r)
						//TODO: SEND EXPORT

						w.Show()
						other_window.Close()
						// we are almost done
					}, other_window)
				file_Dialog.Resize(fyne.NewSize(600, 400))
				file_Dialog.Show()
				// Show file selection dialog.

			}),

			widget.NewButton("Update", func() {
				if !work {
					return
				}
				//TODO: SEND LIST
				list = append(list, "Un bouton Clicked")
				fmt.Println("Update was clicked!")
				renderListScreen(a, w, list)
			}),
			widget.NewButton("See More", func() {
				if !work {
					return
				}
				fmt.Println("See More Clicked")
				if current_selected >= 0 {
					work = false
					fmt.Println("You Select :", list[current_selected])

					grey := color.NRGBA{R: 0xaa, B: 0xaa, G: 0xaa, A: 0xaa}

					canvas.NewColorRGBAAnimation(color.Transparent, grey, time.Second*2, func(c color.Color) {
						rect.FillColor = c
						rect.StrokeColor = c
						canvas.Refresh(rect)
					}).Start()
					//TODO: DATA_DL
				} else {
					fmt.Println("You Select no one !")
				}

			}),
		),

		nil, // Right
		nil, // Left

		// the rest will take all the rest of the space
		element,
	)
	w.SetContent(container.NewStack(content, rect))
}

func main() {
	//list := []string{"Jean", "Pierre", "Nico", "Jch", "blabla", "coucou", "ohohoh", "Gertrude", "Janine", "Odette", "Robert", "Un autre gars", "Quelqu'un", "Auguste", "Plus d'idée"}
	a := app.New()
	w := a.NewWindow("P2PSHARE")

	w.Resize(fyne.NewSize(400, 600))

	//renderListScreen(a, w, list)
	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			switch id {
			case "":
				return []widget.TreeNodeID{"a", "b", "c"}
			case "a":
				return []widget.TreeNodeID{"a1", "a2"}

			case "a2":
				return []widget.TreeNodeID{"a21", "a22", "a23"}

			case "a22":
				return []widget.TreeNodeID{"a221", "a222", "a223", "224"}
			}
			return []string{}
		},
		func(id widget.TreeNodeID) bool {
			return id == "" || id == "a" || id == "a2" || id == "a22"
		},
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabel("Branch template")
			}
			return widget.NewLabel("Leaf template")
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			text := id
			if branch {
				text += " (branch)"
			}
			o.(*widget.Label).SetText(text)
		})

	w.SetContent(tree)
	w.ShowAndRun()
}
