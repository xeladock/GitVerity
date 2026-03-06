package main

import (
	"time"

	"GitVersity/vars"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func main() {
	myApp := app.NewWithID("config-comparator")
	myApp.Settings().SetTheme(theme.LightTheme())
	w := myApp.NewWindow("Сравнение конфигураций сетевых устройств")

	left := vars.NewSidePanel("Левая конфигурация")
	right := vars.NewSidePanel("Правая конфигурация")

	left.OtherPanel = right
	right.OtherPanel = left
	toolbar := vars.CreateToolbar(left, right, w)
	//прокрутка до уровня.
	go func() {
		var lastLeft, lastRight fyne.Position
		for {
			time.Sleep(500 * time.Millisecond)
			if left.Content == nil || right.Content == nil {
				continue
			}
			changed := false

			if left.Content.Offset != lastLeft {
				right.Content.Offset = left.Content.Offset
				fyne.Do(func() {
					right.Content.Refresh()
				})
				//right.Content.Refresh()
				lastLeft = left.Content.Offset
				lastRight = left.Content.Offset
				changed = true
			}
			if right.Content.Offset != lastRight {
				left.Content.Offset = right.Content.Offset
				fyne.Do(func() {
					left.Content.Refresh()
				})
				//left.Content.Refresh()
				lastRight = right.Content.Offset
				lastLeft = right.Content.Offset
				changed = true
			}
			if changed {
				// можно добавить логику, чтобы не спамить Refresh лишний раз
			}
		}
	}()

	// Синхронизированная прокрутка
	left.Content.Offset = right.Content.Offset
	fyne.Do(func() {
		left.Content.Refresh()
	})
	//left.Content.Refresh()

	right.Content.Offset = right.Content.Offset
	fyne.Do(func() {
		right.Content.Refresh()
	})
	leftScroll := container.NewVScroll(left.Container)
	rightScroll := container.NewVScroll(right.Container)

	leftScroll.SetMinSize(fyne.NewSize(400, 0))
	rightScroll.SetMinSize(fyne.NewSize(400, 0))

	Content := container.NewHSplit(leftScroll, rightScroll)
	Content.SetOffset(0.5)

	//Сontent.SetOffsetUpdate(func(offset float32) {
	//	if offset < 0.4 || offset > 0.6 {
	//		Сontent.SetOffset(0.5)
	//	}
	//})

	mainContainer := container.NewBorder(
		toolbar,
		nil, nil, nil,
		Content,
	)

	w.SetContent(mainContainer)

	minContent := mainContainer.MinSize()

	// Делаем минимальный размер окна не меньше заданного тобой порога
	forcedMin := fyne.NewSize(
		fyne.Max(minContent.Width, 900),  // ширина не меньше 900
		fyne.Max(minContent.Height, 650), // высота не меньше 650
	)

	// Устанавливаем начальный размер (чуть больше минимума, чтобы выглядело комфортно)
	w.Resize(fyne.NewSize(forcedMin.Width*1.1, forcedMin.Height*1.1))
	w.SetMaster()
	w.ShowAndRun()
}

//1010
// upDateDateOptions обновляет списки дат в обоих Select'ах,
// исключая уже выбранную дату в противоположной панели

// 1011
//type SidePanel struct {
//	title        string
//	container    fyne.CanvasObject
//	Content      *container.Scroll
//	Text         *widget.RichText
//	Date         string
//	Group        string // ← было Vendor, теперь ЦОД / ЛВС
//	Vendor       string // ← новое: Cisco / Juniper / ...
//	File         string
//	DateSelect   *widget.Select
//	GroupSelect  *widget.Select // ← новое поле
//	VendorSelect *widget.Select
//	FileSelect   *widget.Select
//	//status       *widget.Label
//	otherPanel *SidePanel
//	Syncing    bool
//}

// trySyncFromDate1ToDate2 — пытается подтянуть группу → вендор → файл из Дата 1 в Дата 2
// Вызывается только при смене даты в Дата 2
