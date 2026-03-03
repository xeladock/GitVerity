package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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

var minWindowSize = fyne.NewSize(900, 650)

type ConfigChoice struct {
	Date   string
	Vendor string
	File   string
	Path   string
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("=======================================")
			log.Printf("ПАНИКА: %v", r)
			log.Printf("Stack trace:\n%s", debug.Stack())
			log.Printf("=======================================")

			// Если окно уже создано, можно показать диалог (опционально)
			// dialog.ShowError(fmt.Errorf("Произошла критическая ошибка: %v", r), w)
		}
	}()
	myApp := app.NewWithID("config-comparator")
	myApp.Settings().SetTheme(theme.LightTheme())
	w := myApp.NewWindow("Сравнение конфигураций сетевых устройств")

	left := newSidePanel("Левая конфигурация")
	right := newSidePanel("Правая конфигурация")

	left.otherPanel = right
	right.otherPanel = left

	go func() {
		var lastLeft, lastRight fyne.Position
		for {
			time.Sleep(500 * time.Millisecond)
			if left.content == nil || right.content == nil {
				continue
			}
			changed := false

			if left.content.Offset != lastLeft {
				right.content.Offset = left.content.Offset
				fyne.Do(func() {
					right.content.Refresh()
				})
				//right.content.Refresh()
				lastLeft = left.content.Offset
				lastRight = left.content.Offset
				changed = true
			}
			if right.content.Offset != lastRight {
				left.content.Offset = right.content.Offset
				fyne.Do(func() {
					left.content.Refresh()
				})
				//left.content.Refresh()
				lastRight = right.content.Offset
				lastLeft = right.content.Offset
				changed = true
			}
			if changed {
				// можно добавить логику, чтобы не спамить Refresh лишний раз
			}
		}
	}()

	// Синхронизированная прокрутка
	left.content.Offset = right.content.Offset
	fyne.Do(func() {
		left.content.Refresh()
	})
	//left.content.Refresh()

	right.content.Offset = right.content.Offset
	fyne.Do(func() {
		right.content.Refresh()
	})
	//right.content.Refresh()
	// Основной layout
	content := container.NewHSplit(
		left.container,
		right.container,
	)
	content.SetOffset(0.5)

	toolbar := createToolbar(left, right, w)

	mainContainer := container.NewBorder(
		toolbar,
		nil, nil, nil,
		content,
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
	//w.Resize(minWindowSize)
	//w.SetMinSize(minWindowSize)
	w.SetMaster()

	w.ShowAndRun()
}
func createToolbar(left, right *SidePanel, w fyne.Window) fyne.CanvasObject {
	dates := loadAvailableDates()
	if len(dates) == 0 {
		return widget.NewLabel("Не найдено ни одной даты")
	}

	date1Select := widget.NewSelect(dates, func(s string) {
		left.date = s
		left.updateGroups()
	})
	date2Select := widget.NewSelect(dates, func(s string) {
		right.date = s
		right.updateGroups()
	})

	date1Select.PlaceHolder = "Выберите дату..."
	date2Select.PlaceHolder = "Выберите дату..."

	// ────────────────────────────────────────
	// Новый Select для группы (ЦОД / ЛВС)
	group1 := widget.NewSelect([]string{}, func(s string) {
		left.group = s
		left.updateVendors()
	})
	group2 := widget.NewSelect([]string{}, func(s string) {
		right.group = s
		right.updateVendors()
	})

	// ────────────────────────────────────────
	// Select для вендора (Cisco, Juniper...)
	vendor1 := widget.NewSelect([]string{}, func(s string) {
		left.vendor = s
		left.updateFiles()
	})
	vendor2 := widget.NewSelect([]string{}, func(s string) {
		right.vendor = s
		right.updateFiles()
	})

	file1 := widget.NewSelect([]string{}, func(selected string) {
		if selected == left.file { // ничего не изменилось — выходим
			return
		}
		left.file = selected
		left.loadAndCompare()
		if right != nil && left.file != "" && !left.syncing {
			left.syncing = true
			defer func() { left.syncing = false }()
			left.trySyncFileTo(right)
		}
	})

	file2 := widget.NewSelect([]string{}, func(selected string) {
		if selected == right.file { // ничего не изменилось — выходим
			return
		}
		right.file = selected
		right.loadAndCompare()
		if left != nil && right.file != "" && !right.syncing {
			right.syncing = true
			defer func() { right.syncing = false }()
			right.trySyncFileTo(left)
		}
	})

	compareBtn := widget.NewButtonWithIcon("Сравнить", theme.ViewRefreshIcon(), func() {
		left.loadAndCompare()
		right.loadAndCompare()
	})

	status := widget.NewLabel("Готов")

	toolbar := container.NewGridWithColumns(9, // стало больше колонок
		widget.NewLabel("Дата 1:"), date1Select,
		widget.NewLabel("Группа:"), group1,
		widget.NewLabel("Вендор:"), vendor1,
		widget.NewLabel("Файл:"), file1,
		layout.NewSpacer(),
		widget.NewLabel("Дата 2:"), date2Select,
		widget.NewLabel("Группа:"), group2,
		widget.NewLabel("Вендор:"), vendor2,
		widget.NewLabel("Файл:"), file2,
		layout.NewSpacer(),
		compareBtn,
		status,
	)

	// Связываем
	left.dateSelect = date1Select
	left.groupSelect = group1 // новое
	left.vendorSelect = vendor1
	left.fileSelect = file1
	left.status = status

	right.dateSelect = date2Select
	right.groupSelect = group2 // новое
	right.vendorSelect = vendor2
	right.fileSelect = file2
	right.status = status

	return toolbar
}

type SidePanel struct {
	title        string
	container    fyne.CanvasObject
	content      *container.Scroll
	text         *widget.RichText
	date         string
	group        string // ← было vendor, теперь ЦОД / ЛВС
	vendor       string // ← новое: Cisco / Juniper / ...
	file         string
	dateSelect   *widget.Select
	groupSelect  *widget.Select // ← новое поле
	vendorSelect *widget.Select
	fileSelect   *widget.Select
	status       *widget.Label
	otherPanel   *SidePanel
	syncing      bool
}

func (p *SidePanel) trySyncFileTo(other *SidePanel) {
	if other == nil || p.file == "" || p.syncing || other.syncing {
		return
	}

	if other.file == p.file { // уже тот же файл — ничего не делаем
		return
	}

	targetPath := filepath.Join(rootDir, other.date, "config_files_clear", other.group, other.vendor, p.file)

	if _, err := os.Stat(targetPath); err == nil {
		other.syncing = true
		defer func() { other.syncing = false }()
		other.file = p.file
		other.fileSelect.SetSelected(p.file) // это вызовет callback, но с защитой
		other.loadAndCompare()
	} else {
		if other.date != "" && other.group != "" && other.vendor != "" {
			other.file = ""                  // очищаем файл
			other.fileSelect.ClearSelected() // сбрасываем выбор в Select
			other.text.ParseMarkdown(
				fmt.Sprintf("**Файл отсутствует** на дату **%s**\n\n"+
					"Имя файла: `%s`\n"+
					"Группа: %s\nВендор: %s\n"+
					"Путь, где искали: `%s`\n"+
					"Ошибка: %v",
					other.date, p.file, other.group, other.vendor, targetPath, err),
			)
			other.text.Refresh()
		}
	}
}

func newSidePanel(title string) *SidePanel {

	p := &SidePanel{title: title}
	//if p.text == nil {
	//	panic("widget.NewRichTextFromMarkdown вернул nil") // для отладки
	//}
	p.text = widget.NewRichText() // ← пустой, без Markdown
	if p.text == nil {
		panic("widget.NewRichTextFromMarkdown вернул nil") // для отладки
	}
	//p.text.Wrapping = fyne.TextWrapWord
	p.content = container.NewVScroll(p.text)
	p.content.SetMinSize(fyne.NewSize(400, 500))

	header := widget.NewLabelWithStyle(
		fmt.Sprintf("  %s", title),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	pathLabel := widget.NewLabel("Путь: —")
	p.container = container.NewBorder(
		container.NewHBox(header, layout.NewSpacer(), pathLabel),
		nil, nil, nil,
		p.content,
	)

	return p
}
func (p *SidePanel) updateGroups() {
	if p.date == "" || p.groupSelect == nil {
		log.Println("updateGroups: дата пустая или groupSelect nil")
		return
	}

	groups := loadGroupsForDate(p.date)
	sort.Strings(groups)
	p.groupSelect.Options = groups

	// Если ранее выбранная группа больше не существует → сбрасываем
	if p.group != "" && !contains(groups, p.group) {
		p.group = ""
		p.groupSelect.ClearSelected()
	}

	// НЕ выбираем автоматически первую группу!
	// p.group = groups[0]; p.groupSelect.SetSelected(p.group) ← УДАЛИТЬ

	p.updateVendors()
}

func (p *SidePanel) updateVendors() {
	if p.date == "" || p.group == "" || p.vendorSelect == nil {
		return
	}

	vendors := loadVendorsForDateGroup(p.date, p.group)
	sort.Strings(vendors)
	p.vendorSelect.Options = vendors

	if p.vendor != "" && !contains(vendors, p.vendor) {
		p.vendor = ""
		p.vendorSelect.ClearSelected()
	}

	// НЕ выбираем автоматически первого вендора!
	// p.vendor = vendors[0]; p.vendorSelect.SetSelected(p.vendor) ← УДАЛИТЬ

	p.updateFiles()
}

func (p *SidePanel) updateFiles() {
	if p.date == "" || p.group == "" || p.vendor == "" || p.fileSelect == nil {
		return
	}
	files := loadFilesForDateGroupVendor(p.date, p.group, p.vendor)
	sort.Strings(files)
	p.fileSelect.Options = files

	if p.file != "" && !contains(files, p.file) {
		p.file = ""
		p.fileSelect.ClearSelected()
		p.text.ParseMarkdown("**Выбранный ранее файл больше недоступен** в этой папке")
		p.text.Refresh()
	}

	// НЕ выбираем автоматически первый файл!
	// if len(files) > 0 && p.file == "" {
	//     p.file = files[0]
	//     p.fileSelect.SetSelected(p.file)
	// } ← УДАЛИТЬ ЭТОТ БЛОК

	p.loadAndCompare()
}

func (p *SidePanel) loadAndCompare() {
	if p.text == nil {
		log.Println("p.text is nil in loadAndCompare")
		return
	}
	if p.date == "" || p.vendor == "" || p.file == "" {
		p.text.ParseMarkdown("**Выберите дату, вендора и файл**")
		return
	}
	path := filepath.Join(rootDir, p.date, "config_files_clear", p.group, p.vendor, p.file)
	println("path в loadAndCompare")
	//println(path)
	//path := filepath.Join(rootDir, p.date, p.vendor, p.file)
	data, err := os.ReadFile(path)
	if err != nil {
		p.text.ParseMarkdown(fmt.Sprintf("**Ошибка чтения файла:**\n```\n%s\n```", err))
		return
	}

	content := string(data)
	p.text.ParseMarkdown("```\n" + content + "\n```")

	// Если есть вторая панель и выбран файл — делаем diff
	if p.otherPanel != nil && p.otherPanel.file != "" && p.file != "" {
		compareTwoPanels(p, p.otherPanel)
	}
}

// ---------------------------------------------------------------
// Вспомогательные функции
// ---------------------------------------------------------------

func loadGroupsForDate(date string) []string {
	base := filepath.Join(rootDir, date, "config_files_clear")
	entries, err := os.ReadDir(base)
	if err != nil {
		log.Printf("Нет доступа к %s: %v", base, err)
		return nil
	}
	var groups []string
	for _, e := range entries {
		if e.IsDir() {
			groups = append(groups, e.Name())
		}
	}
	return groups
}

func loadVendorsForDateGroup(date, group string) []string {
	path := filepath.Join(rootDir, date, "config_files_clear", group)
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Нет доступа к %s: %v", path, err)
		return nil
	}
	var vendors []string
	for _, e := range entries {
		if e.IsDir() {
			vendors = append(vendors, e.Name())
		}
	}
	return vendors
}

func loadFilesForDateGroupVendor(date, group, vendor string) []string {
	path := filepath.Join(rootDir, date, "config_files_clear", group, vendor)
	log.Printf("Сканирую файлы в: %s", path) // для отладки

	var files []string

	entries, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Ошибка чтения %s: %v", path, err)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	log.Printf("Найдено файлов: %d → %v", len(files), files) // для отладки

	return files
}

func loadAvailableDates() []string {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		log.Printf("Ошибка чтения текущей директории: %v", err)
		return nil
	}

	var dates []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if looksLikeDate(name) {
			// Проверяем, существует ли внутри config_files_clear
			inner := filepath.Join(rootDir, name, "config_files_clear")
			if fi, err := os.Stat(inner); err == nil && fi.IsDir() {
				dates = append(dates, name)
			}
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates))) // новые даты сверху
	return dates
}

func loadVendorsForDate(date string) []string {
	//base := filepath.Join(rootDir, p.date, "config_files_clear", p.vendor, p.file)
	base := filepath.Join(rootDir, date, "config_files_clear")
	entries, err := os.ReadDir(base)
	if err != nil {
		log.Printf("Нет доступа к %s: %v", base, err)
		return nil
	}

	var groups []string // ЦОД, ЛВС и т.д.
	for _, e := range entries {
		if e.IsDir() {
			groups = append(groups, e.Name())
		}
	}
	sort.Strings(groups)
	return groups
}

func loadFilesForDateVendor(date, group string) []string {
	path := filepath.Join(rootDir, date, "config_files_clear", group)
	var files []string

	// Предполагаем, что внутри группы лежат папки вендоров (Cisco, Juniper, ...)
	vendorDirs, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, vd := range vendorDirs {
		if !vd.IsDir() {
			continue
		}
		vendorPath := filepath.Join(path, vd.Name())
		fis, err := os.ReadDir(vendorPath)
		if err != nil {
			continue
		}
		for _, fi := range fis {
			if !fi.IsDir() {
				// относительный путь от группы, например Cisco/config1.txt
				rel := filepath.Join(vd.Name(), fi.Name())
				files = append(files, rel)
			}
		}
	}

	sort.Strings(files)
	return files
}
func looksLikeDate(s string) bool {
	if len(s) != 8 {
		return false
	}
	if s[2] != '-' || s[5] != '-' {
		return false
	}
	// можно ещё проверить цифры, но пока достаточно
	return true
}

func compareTwoPanels(a, b *SidePanel) {
	if a.file == "" || b.file == "" {
		return
	}

	pathA := filepath.Join(rootDir, a.date, "config_files_clear", a.group, a.vendor, a.file)
	pathB := filepath.Join(rootDir, b.date, "config_files_clear", b.group, b.vendor, b.file)

	dataA, errA := os.ReadFile(pathA)
	dataB, errB := os.ReadFile(pathB)
	if errA != nil || errB != nil {
		a.status.SetText("Ошибка чтения одного из файлов")
		return
	}

	textA := string(dataA)
	textB := string(dataB)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(textA, textB, false)
	dmp.DiffCleanupSemantic(diffs)

	var leftSegments, rightSegments []widget.RichTextSegment

	for _, diff := range diffs {
		text := diff.Text

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
			Text:  text,
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

	a.text.Segments = leftSegments
	b.text.Segments = rightSegments

	a.text.Refresh()
	b.text.Refresh()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
