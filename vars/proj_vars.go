package vars

import (
	//"GitVersity/btn_add"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	rootDir = "."
	//minWindowSize = fyne.NewSize(900, 650)
)

type SidePanel struct {
	Title        string
	Container    fyne.CanvasObject
	Content      *container.Scroll
	Text         *widget.RichText
	Date         string
	Group        string
	Vendor       string
	File         string
	DateSelect   *widget.Select
	GroupSelect  *widget.Select
	VendorSelect *widget.Select
	FileSelect   *widget.Select
	OtherPanel   *SidePanel
	Syncing      bool
}

func NewSidePanel(title string) *SidePanel {
	p := &SidePanel{Title: title}

	p.Text = widget.NewRichText() // ← пустой, без Markdown
	if p.Text == nil {
		panic("widget.NewRichText вернул nil")
	}
	// p.Text.Wrapping = fyne.TextWrapWord  // если нужно
	p.Text.Wrapping = fyne.TextWrapOff

	p.Content = container.NewVScroll(p.Text)
	//p.Content.SetMinSize(fyne.NewSize(400, 500))

	header := widget.NewLabelWithStyle(
		fmt.Sprintf("  %s", title),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	pathLabel := widget.NewLabel("Путь: —")

	p.Container = container.NewBorder(
		container.NewHBox(header, layout.NewSpacer(), pathLabel),
		nil, nil, nil,
		p.Content,
	)

	// Важно! Здесь НЕ создаём Select'ы — они создаются в createToolbar
	// и присваиваются позже через поля с большой буквы
	//log.Println("NewSidePanel создан для", title)
	return p
}

func (p *SidePanel) UpDateGroups() {
	if p.Date == "" || p.GroupSelect == nil {
		log.Println("upDateGroups: дата пустая или GroupSelect nil")
		return
	}

	Groups := LoadGroupsForDate(p.Date)
	sort.Strings(Groups)
	p.GroupSelect.Options = Groups

	// Если ранее выбранная группа больше не существует → сбрасываем
	if p.Group != "" && !Contains(Groups, p.Group) {
		p.Group = ""
		p.GroupSelect.ClearSelected()
	}

	// НЕ выбираем автоматически первую группу!
	// p.Group = Groups[0]; p.GroupSelect.SetSelected(p.Group) ← УДАЛИТЬ

	p.UpDateVendors()

}

func (p *SidePanel) UpDateVendors() {
	if p.Date == "" || p.Group == "" || p.VendorSelect == nil {
		return
	}

	Vendors := LoadVendorsForDateGroup(p.Date, p.Group)
	sort.Strings(Vendors)
	p.VendorSelect.Options = Vendors

	if p.Vendor != "" && !Contains(Vendors, p.Vendor) {
		p.Vendor = ""
		p.VendorSelect.ClearSelected()
		p.FileSelect.ClearSelected()
	}

	// НЕ выбираем автоматически первого вендора!
	// p.Vendor = Vendors[0]; p.VendorSelect.SetSelected(p.Vendor) ← УДАЛИТЬ

	p.UpDateFiles()

}

func (p *SidePanel) UpDateFiles() {
	if p.Date == "" || p.Group == "" || p.Vendor == "" || p.FileSelect == nil {
		return
	}
	Files := LoadFilesForDateGroupVendor(p.Date, p.Group, p.Vendor)
	sort.Strings(Files)
	p.FileSelect.Options = Files

	if p.File != "" && !Contains(Files, p.File) {
		p.File = ""
		p.FileSelect.ClearSelected()
		p.Text.ParseMarkdown("**Выбранный ранее файл больше недоступен** в этой папке")
		p.Text.Refresh()
	}

	// НЕ выбираем автоматически первый файл!
	// if len(Files) > 0 && p.File == "" {
	//     p.File = Files[0]
	//     p.FileSelect.SetSelected(p.File)
	// } ← УДАЛИТЬ ЭТОТ БЛОК

	p.LoadAndCompare()
}
func (p *SidePanel) LoadAndCompare() {
	if p.Text == nil {
		log.Println("p.Text is nil in LoadAndCompare")
		return
	}
	//if p.Date == "" || p.Vendor == "" || p.File == "" {
	//	p.Text.ParseMarkdown("**Выберите дату, вендора и файл**")
	//	return
	//}
	path := filepath.Join(rootDir, p.Date, "config_files_clear", p.Group, p.Vendor, p.File)
	//println("path в LoadAndCompare")
	//println(path)
	//path := Filepath.Join(rootDir, p.Date, p.Vendor, p.File)
	data, _ := os.ReadFile(path)
	//if err != nil {
	//	//p.Text.ParseMarkdown(fmt.Sprintf("**Ошибка чтения файла:**\n```\n%s\n```", err))
	//	//return
	//}

	Content := string(data)
	fyne.Do(func() {
		p.Text.ParseMarkdown("```\n" + Content + "\n```")
	})

	// Если есть вторая панель и выбран файл — делаем diff
	if p.OtherPanel != nil && p.OtherPanel.File != "" && p.File != "" {
		compareTwoPanels(p, p.OtherPanel)
	}
}

func (p *SidePanel) TrySyncFileTo(other *SidePanel) {
	if other == nil || p.File == "" || p.Syncing || other.Syncing {
		return
	}

	if other.File == p.File { // уже тот же файл — ничего не делаем
		return
	}

	targetPath := filepath.Join(rootDir, other.Date, "config_files_clear", other.Group, other.Vendor, p.File)

	if _, err := os.Stat(targetPath); err == nil {
		other.Syncing = true
		defer func() { other.Syncing = false }()
		other.File = p.File
		other.FileSelect.SetSelected(p.File) // это вызовет callback, но с защитой
		other.LoadAndCompare()
	} else {
		if other.Date != "" && other.Group != "" && other.Vendor != "" {
			other.File = ""                  // очищаем файл
			other.FileSelect.ClearSelected() // сбрасываем выбор в Select
			other.Text.ParseMarkdown(
				fmt.Sprintf("**Файл отсутствует** на дату **%s**\n\n"+
					"Имя файла: `%s`\n"+
					"Группа: %s\nВендор: %s\n"+
					"Путь, где искали: `%s`\n"+
					"Ошибка: %v",
					other.Date, p.File, other.Group, other.Vendor, targetPath, err),
			)
			other.Text.Refresh()
		}
	}
}
func UpDateDateOptions(left, right *SidePanel) {
	allDates := LoadAvailableDates()
	if len(allDates) == 0 {
		return
	}

	// Для Дата 1: исключаем дату, выбранную в Дата 2 (если она выбрана)
	Date1Options := make([]string, 0, len(allDates))
	if right.Date != "" {
		for _, d := range allDates {
			if d != right.Date {
				Date1Options = append(Date1Options, d)
			}
		}
	} else {
		Date1Options = allDates
	}
	left.DateSelect.Options = Date1Options

	// Если текущая дата левой панели исчезла из списка — сбрасываем
	if left.Date != "" && !Contains(Date1Options, left.Date) {
		left.Date = ""
		left.DateSelect.ClearSelected()
		left.ResetLowerFields() // если у тебя есть такая функция сброса
	}

	// Для Дата 2: исключаем дату из Дата 1
	Date2Options := make([]string, 0, len(allDates))
	if left.Date != "" {
		for _, d := range allDates {
			if d != left.Date {
				Date2Options = append(Date2Options, d)
			}
		}
	} else {
		Date2Options = allDates
	}
	right.DateSelect.Options = Date2Options

	if right.Date != "" && !Contains(Date2Options, right.Date) {
		right.Date = ""
		right.DateSelect.ClearSelected()
		right.ResetLowerFields()
	}

	// Обновляем UI
	left.DateSelect.Refresh()
	right.DateSelect.Refresh()
}

func LoadFilesForDateGroupVendor(Date, Group, Vendor string) []string {
	path := filepath.Join(rootDir, Date, "config_files_clear", Group, Vendor)
	//log.Printf("Сканирую файлы в: %s", path) // для отладки

	var Files []string

	entries, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Ошибка чтения %s: %v", path, err)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			Files = append(Files, entry.Name())
		}
	}

	sort.Strings(Files)
	//log.Printf("Найдено файлов: %d → %v", len(Files), Files) // для отладки

	return Files
}
func compareTwoPanels(a, b *SidePanel) {
	if a.File == "" || b.File == "" {
		return
	}

	pathA := filepath.Join(rootDir, a.Date, "config_files_clear", a.Group, a.Vendor, a.File)
	pathB := filepath.Join(rootDir, b.Date, "config_files_clear", b.Group, b.Vendor, b.File)

	dataA, errA := os.ReadFile(pathA)
	dataB, errB := os.ReadFile(pathB)
	if errA != nil || errB != nil {
		//a.status.SetText("Ошибка чтения одного из файлов")
		return
	}

	TextA := string(dataA)
	TextB := string(dataB)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(TextA, TextB, false)
	dmp.DiffCleanupSemantic(diffs)

	var leftSegments, rightSegments []widget.RichTextSegment

	for _, diff := range diffs {
		Text := diff.Text

		var style widget.RichTextStyle

		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			style = widget.RichTextStyle{} // обычный текст

		case diffmatchpatch.DiffDelete:
			style = widget.RichTextStyle{
				//TextStyle: fyne.TextStyle{Bold: true}, // оставляем жирный, если хочешь
				// ColorName убрали → будет обычный цвет
			}

		case diffmatchpatch.DiffInsert:
			style = widget.RichTextStyle{
				//TextStyle: fyne.TextStyle{Bold: true},
				// ColorName убрали
			}
		}

		seg := &widget.TextSegment{
			Text:  Text,
			Style: style,
		}

		if diff.Type == diffmatchpatch.DiffEqual {
			leftSegments = append(leftSegments, seg)
			rightSegments = append(rightSegments, seg)
		} else if diff.Type == diffmatchpatch.DiffDelete {
			leftSegments = append(leftSegments, seg)
			rightSegments = append(rightSegments, &widget.TextSegment{Text: ""})
		} else if diff.Type == diffmatchpatch.DiffInsert {
			leftSegments = append(leftSegments, &widget.TextSegment{Text: ""})
			rightSegments = append(rightSegments, seg)
		}
	}

	fyne.Do(func() {
		a.Text.Segments = leftSegments
		b.Text.Segments = rightSegments

		a.Text.Refresh()
		b.Text.Refresh()
	})

}
func LoadGroupsForDate(Date string) []string {
	base := filepath.Join(rootDir, Date, "config_files_clear")
	entries, err := os.ReadDir(base)
	if err != nil {
		log.Printf("Нет доступа к %s: %v", base, err)
		return nil
	}
	var Groups []string
	for _, e := range entries {
		if e.IsDir() {
			Groups = append(Groups, e.Name())
		}
	}
	return Groups
}

func LoadVendorsForDateGroup(Date, Group string) []string {
	path := filepath.Join(rootDir, Date, "config_files_clear", Group)
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Нет доступа к %s: %v", path, err)
		return nil
	}
	var Vendors []string
	for _, e := range entries {
		if e.IsDir() {
			Vendors = append(Vendors, e.Name())
		}
	}
	return Vendors
}

func LoadAvailableDates() []string {
	entries, err := os.ReadDir(rootDir)
	//log.Printf("Текущая рабочая директория: %s", entries)
	//log.Printf("Сканирую даты в папке: %s", rootDir)
	if err != nil {
		log.Printf("Ошибка чтения текущей директории: %v", err)
		return nil
	}

	var Dates []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if LooksLikeDate(name) {
			// Проверяем, существует ли внутри config_files_clear
			inner := filepath.Join(rootDir, name, "config_files_clear")
			//fmt.Println(inner)
			if fi, err := os.Stat(inner); err == nil && fi.IsDir() {
				Dates = append(Dates, name)
			}
			//fmt.Println(Dates)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(Dates))) // новые даты сверху
	return Dates
}

func LooksLikeDate(s string) bool {
	if len(s) != 8 {
		return false
	}
	if s[2] != '-' || s[5] != '-' {
		return false
	}
	// можно ещё проверить цифры, но пока достаточно
	return true
}

//	func (p *SidePanel) File() string {
//		return p.file
//	}
//
//	func (p *SidePanel) SetFile(f string) {
//		p.file = f
//	}
//
//	func (p *SidePanel) Group() string {
//		return p.group
//	}
//
//	func (p *SidePanel) SetGroup(g string) {
//		p.group = g
//	}
//
// func (p *SidePanel) Vendor() string {}
func (p *SidePanel) ResetLowerFields() {
	p.Group = ""
	if p.GroupSelect != nil {
		p.GroupSelect.ClearSelected()
	}

	p.Vendor = ""
	if p.VendorSelect != nil {
		p.VendorSelect.ClearSelected()
	}

	p.File = ""
	if p.FileSelect != nil {
		p.FileSelect.ClearSelected()
	}

	// Очищаем текст в панели
	p.Text.ParseMarkdown("**Дата изменена. Выберите группу, вендора и файл заново.**")
	p.Text.Refresh()
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
func TrySyncFromDate1ToDate2(left, right *SidePanel) {
	println("зашли в TrySyncFromDate1ToDate2")
	if left == nil || right == nil {
		log.Println("trySync: панели nil")
		return
	}

	if right.GroupSelect == nil {
		log.Println("trySync: right Group")
		return
	}
	if right.VendorSelect == nil {
		log.Println("trySync: right Vendor")
		return
	}

	if right.FileSelect == nil {
		log.Println("trySync: right File")
		return
	}
	//if right.status == nil {
	//	log.Println("trySync: right status")
	//	return
	//}

	if left.Group == "" || left.Vendor == "" || left.File == "" {
		log.Println("trySync: в Дата 1 не всё выбрано Group")
		return
	}
	println("прошли циклы проверки")
	//log.Println("trySyncFromDate1ToDate2: нормальный вызов, начинаем подтягивание")

	// 1. Группа
	Groups := LoadGroupsForDate(right.Date)
	if !Contains(Groups, left.Group) {
		right.Text.ParseMarkdown(fmt.Sprintf(
			"**Для даты %s нет группы '%s'**\n"+
				"Выберите группу вручную.",
			right.Date, left.Group))
		right.Text.Refresh()
		right.Group = ""
		right.GroupSelect.ClearSelected()
		//right.status.SetText("Группа не найдена")
		return
	}
	println("прошли циклы проверки2")
	right.Group = left.Group
	right.GroupSelect.SetSelected(right.Group)
	right.UpDateVendors()

	// 2. Вендор
	Vendors := LoadVendorsForDateGroup(right.Date, right.Group)
	if !Contains(Vendors, left.Vendor) {
		right.Text.ParseMarkdown(fmt.Sprintf(
			"**Для даты %s в группе '%s' нет вендора '%s'**\n"+
				"Выберите вендор вручную.",
			right.Date, right.Group, left.Vendor))
		right.Text.Refresh()
		right.Vendor = ""
		right.VendorSelect.ClearSelected()
		//right.status.SetText("Вендор не найден")
		return
	}
	println("прошли циклы проверки3")
	right.Vendor = left.Vendor
	right.VendorSelect.SetSelected(right.Vendor)
	right.UpDateFiles()

	// 3. Файл
	Files := LoadFilesForDateGroupVendor(right.Date, right.Group, right.Vendor)
	println(Files)
	println(left.File)
	if !Contains(Files, left.File) {
		println("вошли в цикл")

		message := fmt.Sprintf(
			"**В этой папке нет файла '%s'**\n",
			left.File,
		)

		println(message) // для консоли

		fyne.Do(func() {
			right.Text.ParseMarkdown(message)
			right.Text.Refresh()
			println("ParseMarkdown и Refresh вызваны в fyne.Do")
		})

		right.File = ""
		if right.FileSelect != nil {
			right.FileSelect.ClearSelected()
		}
		println("вышли")
		return
	}
	println("прошли циклы проверки4")
	right.File = left.File
	right.FileSelect.SetSelected(right.File)
	right.LoadAndCompare()

	//right.status.SetText("Подтянуто из Дата 1")
}

// 1003

//func NewSidePanel(title string) *SidePanel {
//
//	p := &SidePanel{Title: title}
//
//	p.Text = widget.NewRichText() // ← пустой, без Markdown
//	if p.Text == nil {
//		panic("widget.NewRichTextFromMarkdown вернул nil") // для отладки
//	}
//	//p.Text.Wrapping = fyne.TextWrapWord
//	p.Content = container.NewVScroll(p.Text)
//	p.Content.SetMinSize(fyne.NewSize(400, 500))
//
//	header := widget.NewLabelWithStyle(
//		fmt.Sprintf("  %s", title),
//		fyne.TextAlignLeading,
//		fyne.TextStyle{Bold: true},
//	)
//
//	pathLabel := widget.NewLabel("Путь: —")
//	p.Container = container.NewBorder(
//		container.NewHBox(header, layout.NewSpacer(), pathLabel),
//		nil, nil, nil,
//		p.Content,
//	)
//
//	return p
//}

func CreateToolbar(left, right *SidePanel, w fyne.Window) fyne.CanvasObject {
	resetBtn := ResetButton(left, right)
	Dates := LoadAvailableDates()
	if len(Dates) == 0 {
		return widget.NewLabel("Не найдено ни одной папки")
	}
	//println(Dates)
	//1001
	Date1Select := widget.NewSelect(Dates, func(s string) {

		left.Date = s
		left.UpDateGroups()
		UpDateDateOptions(left, right)
		//TrySyncFromDate1ToDate2(left, right)
	})

	Date2Select := widget.NewSelect(Dates, func(s string) {
		if s == "" {
			return // пропускаем ранний вызов при создании виджета
		}

		right.Date = s
		right.UpDateGroups()
		UpDateDateOptions(left, right)
		TrySyncFromDate1ToDate2(left, right)

		//trySyncFromDate1ToDate2(left, right) // ← возвращаем
	})

	Date1Select.PlaceHolder = "Дата ЛП"
	Date2Select.PlaceHolder = "Дата ПП"

	// ────────────────────────────────────────
	// Новый Select для группы (ЦОД / ЛВС)
	Group1 := widget.NewSelect([]string{}, func(s string) {
		left.Group = s
		left.UpDateVendors()
	})
	Group2 := widget.NewSelect([]string{}, func(s string) {
		right.Group = s
		right.UpDateVendors()
	})
	Group1.PlaceHolder = "Группа ЛП"
	Group2.PlaceHolder = "Группа ПП"
	// ────────────────────────────────────────
	// Select для вендора (Cisco, Juniper...)
	Vendor1 := widget.NewSelect([]string{}, func(s string) {
		left.Vendor = s
		left.UpDateFiles()
	})
	Vendor2 := widget.NewSelect([]string{}, func(s string) {
		right.Vendor = s
		right.UpDateFiles()
	})

	Vendor1.PlaceHolder = "Вендор ЛП"
	Vendor2.PlaceHolder = "Вендор ПП"

	File1 := widget.NewSelect([]string{}, func(selected string) {
		if selected == left.File { // ничего не изменилось — выходим
			return
		}
		left.File = selected
		left.LoadAndCompare()
		if right != nil && left.File != "" && !left.Syncing {
			left.Syncing = true
			defer func() { left.Syncing = false }()
			left.TrySyncFileTo(right)
		}
	})

	File2 := widget.NewSelect([]string{}, func(selected string) {
		if selected == right.File { // ничего не изменилось — выходим
			return
		}
		right.File = selected
		right.LoadAndCompare()
		if left != nil && right.File != "" && !right.Syncing {
			right.Syncing = true
			defer func() { right.Syncing = false }()
			right.TrySyncFileTo(left)
		}
	})

	File1.PlaceHolder = "Файл ЛП"
	File2.PlaceHolder = "Файл ПП"

	compareBtn := widget.NewButtonWithIcon("Сравнить", theme.ViewRefreshIcon(), func() {
		left.LoadAndCompare()
		right.LoadAndCompare()
	})
	syncBtn := widget.NewButton("Подтянуть из Дата 1", func() {
		TrySyncFromDate1ToDate2(left, right)
	})

	toolbar := container.NewVBox(

		container.NewHBox(
			widget.NewLabel("Дата 1:"),
			fixedWidth(Date1Select, 100),
			//hspace(5),

			widget.NewLabel("Группа:"),
			fixedWidth(Group1, 120),
			hspace(5),

			widget.NewLabel("Вендор:"),
			fixedWidth(Vendor1, 150),
			hspace(5),

			widget.NewLabel("Файл:"),
			fixedWidth(File1, 150),

			layout.NewSpacer(),
		),

		container.NewHBox(
			widget.NewLabel("Дата 2:"),
			fixedWidth(Date2Select, 100),
			//hspace(5),

			widget.NewLabel("Группа:"),
			fixedWidth(Group2, 120),
			hspace(5),

			widget.NewLabel("Вендор:"),
			fixedWidth(Vendor2, 150),
			hspace(5),

			widget.NewLabel("Файл:"),
			fixedWidth(File2, 150),

			layout.NewSpacer(),
		),

		container.NewHBox(
			compareBtn,
			syncBtn,
			resetBtn,
			layout.NewSpacer(),
		),
	)

	// Связываем
	left.DateSelect = Date1Select
	left.GroupSelect = Group1 // новое
	left.VendorSelect = Vendor1
	left.FileSelect = File1
	//left.status = status

	right.DateSelect = Date2Select
	right.GroupSelect = Group2 // новое
	right.VendorSelect = Vendor2
	right.FileSelect = File2
	//right.status = status
	//log.Println("CreateToolbar: left.DateSelect =", left.DateSelect != nil)
	//log.Println("CreateToolbar: right.DateSelect =", right.DateSelect != nil)
	//log.Println("CreateToolbar: resetBtn создан")
	//log.Println("CreateToolbar завершён")
	//
	//log.Println("left.DateSelect != nil:", left.DateSelect != nil)
	//log.Println("right.DateSelect != nil:", right.DateSelect != nil)
	//log.Println("left.GroupSelect != nil:", left.GroupSelect != nil)
	//log.Println("toolbar создан, количество объектов:", len(toolbar.Objects))
	//
	//log.Println("CreateToolbar: toolbar создан, объектов:", len(toolbar.Objects))
	//log.Println("left.DateSelect != nil:", left.DateSelect != nil)

	return toolbar
}

func space2(w float32) fyne.CanvasObject {
	return canvas.NewRectangle(color.Transparent)
}
func hspace(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, 1))
	return r
}

func fixedWidth(obj fyne.CanvasObject, w float32) fyne.CanvasObject {
	return container.NewGridWrap(
		fyne.NewSize(w, obj.MinSize().Height),
		obj,
	)
}

func ResetButton(left, right *SidePanel) fyne.CanvasObject {
	return widget.NewButton("Сброс", func() {
		// Сбрасываем левую панель
		left.Date = ""
		left.Group = ""
		left.Vendor = ""
		left.File = ""

		if left.DateSelect != nil {
			left.DateSelect.ClearSelected()
		}
		if left.GroupSelect != nil {
			left.GroupSelect.ClearSelected()
		}
		if left.VendorSelect != nil {
			left.VendorSelect.ClearSelected()
		}
		if left.FileSelect != nil {
			left.FileSelect.ClearSelected()
		}

		// Сбрасываем правую панель
		right.Date = ""
		right.Group = ""
		right.Vendor = ""
		right.File = ""

		if right.DateSelect != nil {
			right.DateSelect.ClearSelected()
		}
		if right.GroupSelect != nil {
			right.GroupSelect.ClearSelected()
		}
		if right.VendorSelect != nil {
			right.VendorSelect.ClearSelected()
		}
		if right.FileSelect != nil {
			right.FileSelect.ClearSelected()
		}

		// Очищаем текст
		left.Text.ParseMarkdown("**Выберите дату, группу, вендора и файл**")
		left.Text.Refresh()
		right.Text.ParseMarkdown("**Выберите дату, группу, вендора и файл**")
		right.Text.Refresh()

		// Обновляем списки дат
		UpDateDateOptions(left, right) // если у тебя есть такая функция
	})
}
