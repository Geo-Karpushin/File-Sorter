package main

import (
	"github.com/dsoprea/go-exif/v3"
    "github.com/dsoprea/go-exif/v3/common"
	"github.com/gotk3/gotk3/gtk"
	"encoding/base64"
	"path/filepath"
	"crypto/sha1"
	"io/ioutil"
	"strconv"
	"strings"
	"net/url"
	"io/fs"
	"time"
	"fmt"
	"log"
	"io"
	"os"
)

var consoleStarted bool
var inputText *gtk.Entry
var mainOpenedWindow *gtk.Window
var determinateFlag bool
var inputPath string
var onDeadMessage string
var sortingMethod bool
var showOnlyName bool
var showDate bool
var needToDelete bool

type Menu struct {
	entries []MenuEntry
}

type MenuEntry struct {
	EntryType int
	Label     string
	Next      *Menu
	Action    func()
}

func (m *Menu) AddEntryNotButton(et int, label string) {
	m.entries = append(m.entries, MenuEntry{
		EntryType: et,
		Label:     label,
		Next:      m,
		Action:    nil,
	})
}

func (m *Menu) AddEntry(et int, label string, next *Menu, act func()) {
	m.entries = append(m.entries, MenuEntry{
		EntryType: et,
		Label:     label,
		Next:      next,
		Action:    act,
	})
}

func (m *Menu) AddEntryWithAction(label string, next *Menu, action func()) {
	if next == nil {
		next = m
	}

	m.entries = append(m.entries, MenuEntry{
		EntryType: 0,
		Label:     label,
		Next:      next,
		Action:    action,
	})
}

func (e MenuEntry) Use() *Menu {
	if e.Action != nil {
		e.Action()
	}

	return e.Next
}

func main() {
	onDeadMessage = ""
	ex, err := os.Executable()
	if err!=nil{
		log.Panic(err)
	}
	inputPath = filepath.Dir(ex)

	gtk.Init(nil)

	initWin()

	gtk.Main()
}

func initWin() {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

	if err != nil {
		log.Fatal(err)
	}

	label, err := gtk.LabelNew(onDeadMessage)

	if err != nil {
		log.Panic(err)
	}

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	box2, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)

	if err != nil {
		log.Panic(err)
	}

	box.SetMarginTop(5)
	box.SetMarginBottom(24)
	box.SetMarginStart(24)
	box.SetMarginEnd(24)

	box.Add(label)
	box.Add(box2)

	var mainMenu, sorting, options, helpChangeSM, helpChangeSN, helpChangeSD, helpChangeNTD Menu

	mainMenu.AddEntryWithAction("Настройки", &options, func() {
		label.SetText("Настройки")
		label.Show()
	})

	mainMenu.AddEntryWithAction("Начать сортировку", &sorting, func() {
		label.SetText("Начало сортировки...")
		startSort(label, box2, []string{}, []string{})
	})

	mainMenu.AddEntryWithAction("Выйти", nil, func() {
		gtk.MainQuit()
	})
	
	options.AddEntryNotButton(1, "Путь")
	
	options.AddEntry(2, getSortingMethod(!sortingMethod), &helpChangeSM, func() {
		var tempMenu Menu
		helpChangeSM = tempMenu
		helpChangeSM.AddEntry(-1,getSortingMethod(sortingMethod), nil,nil)
		sortingMethod=!sortingMethod
	})
	
	options.AddEntry(2,getNeedOnlyShowName(!showOnlyName), &helpChangeSN, func(){
		var tempMenu Menu
		helpChangeSN = tempMenu
		helpChangeSN.AddEntry(-1,getNeedOnlyShowName(showOnlyName), nil,nil)
		showOnlyName=!showOnlyName
	})
	
	options.AddEntry(2,getNeedShowDate(!showDate), &helpChangeSD, func(){
		var tempMenu Menu
		helpChangeSD = tempMenu
		helpChangeSD.AddEntry(-1,getNeedShowDate(showDate), nil,nil)
		showDate=!showDate
	})
	
	options.AddEntry(2,getNeedDelete(!needToDelete), &helpChangeNTD, func(){
		var tempMenu Menu
		helpChangeNTD = tempMenu
		helpChangeNTD.AddEntry(-1,getNeedDelete(needToDelete), nil,nil)
		needToDelete=!needToDelete
	})

	options.AddEntryWithAction("Продолжить", &mainMenu, func() {
		label.SetText("")
	})

	box2.Add(mainMenu.GtkWidget())

	win.Add(box)

	win.Connect("destroy", func() {
		if !determinateFlag {
			gtk.MainQuit()
		} else {
			determinateFlag = false
		}
	})

	mainOpenedWindow = win

	win.ShowAll()
}

func getNeedDelete(in bool) string{
	if in{
		return "Перемещать файл"
	}else{
		return "Копировать файл"
	}
}

func getNeedOnlyShowName(in bool) string{
	if in{
		return "Показывать только имя"
	}else{
		return "Показывать полный путь"
	}
}

func getSortingMethod(in bool) string{
	if in{
		return "Сортировка по дате создания (если доступна)"
	}else{
		return "Сортировка по дате изменения"
	}
}

func getNeedShowDate(in bool) string{
	if in{
		return "Показывать дату"
	}else{
		return "Не показывать дату"
	}
}


func startSort(mlabel *gtk.Label, box *gtk.Box, paths []string, selected []string) {
	list, err := gtk.ListBoxNew()
	tcount := 0

	if err != nil {
		log.Panic(err)
		return
	}
	
	list.SetActivateOnSingleClick(false)
	
	if len(paths)==0{
		err = filepath.Walk(inputPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Println("Ошибка доступа к", path, "-", err)

				onDeadMessage = "Ошибка доступа к " + path

				determinateFlag = true

				mainOpenedWindow.Destroy()

				initWin()
				return nil
			}
			if !info.IsDir() {
				paths = append(paths, path)
				if len(paths) <= 30 {
					y,m,d:=getFileDate(path)
					if showOnlyName{
						path=filepath.Base(path)
					}
					if showDate{
						path+=" - "+d+" "+m+" "+y
					}
					label, err := gtk.LabelNew(path)
					if err != nil {
						log.Panic(err)
					}
					list.Insert(label, len(paths))
				} else {
					tcount += 1
					mlabel.SetText("Обнаружено более 30 файлов. Последние " + strconv.Itoa(tcount) + " файлов не показаны сейчас")
					mlabel.Show()
				}
				return nil
			} else {
				log.Println("-->", path)
			}
			return nil
		})
		if err != nil {
			log.Panic(err)
			return
		}
	}else{
		if len(paths)>30{
			for i:=0; i<31; i++{
				tname:=paths[i]
				if showOnlyName {
					tname=filepath.Base(tname)
				}
				if showDate{
					y,m,d:=getFileDate(paths[i])
					tname += " - "+d+" "+m+" "+y
				}
				label, err := gtk.LabelNew(tname)
				if err != nil {
					log.Panic(err)
				}
				list.Insert(label,i)
			}
			tcount = len(paths)-30
			mlabel.SetText("Обнаружено более 30 файлов. Последние " + strconv.Itoa(tcount) + " файлов не показаны сейчас")
			mlabel.Show()
		}else{
			for i:=0; i<len(paths); i++{
				tname:=paths[i]
				if showOnlyName {
					tname=filepath.Base(tname)
				}
				if showDate{
					y,m,d:=getFileDate(paths[i])
					tname += " - "+d+" "+m+" "+y
				}
				label, err := gtk.LabelNew(tname)
				if err != nil {
					log.Panic(err)
				}
				list.Insert(label,i)
			}
		}
	}

	button, err := gtk.ButtonNewWithLabel("Продолжить")

	if err != nil {
		log.Panic(err)
	}
	
	sbutton, err := gtk.ButtonNewWithLabel("Пропустить")

	if err != nil {
		log.Panic(err)
	}

	cbutton, err := gtk.ButtonNewWithLabel("Отмена")

	if err != nil {
		log.Panic(err)
	}
	
	//locselected:=selected

	button.Connect("clicked", func(h *gtk.Button) {
		anspaths := selected

		list.GetSelectedRows().Foreach(func(child any) {
			chi, _ := child.(*gtk.ListBoxRow)
			log.Println(paths[chi.GetIndex()])
			anspaths = append(anspaths, paths[chi.GetIndex()])
		})
		if(tcount>0){
			box.GetChildren().Foreach(func(child any) {
				btn, _ := child.(*gtk.Widget)
				btn.Destroy()
			})
			startSort(mlabel,box,paths[30:],anspaths)
		}else{
			mlabel.SetText("Сортировка...")
			mlabel.Show()
			box.GetChildren().Foreach(func(child any) {
				btn, _ := child.(*gtk.Widget)
				btn.Destroy()
			})
			defer sortFiles(anspaths, mlabel)
		}
	})
	
	sbutton.Connect("clicked", func(h *gtk.Button) {
		if(tcount>0){
			box.GetChildren().Foreach(func(child any) {
				btn, _ := child.(*gtk.Widget)
				btn.Destroy()
			})
			startSort(mlabel,box,paths[30:],selected)
		}else{
			mlabel.SetText("Сортировка...")
			mlabel.Show()
			box.GetChildren().Foreach(func(child any) {
				btn, _ := child.(*gtk.Widget)
				btn.Destroy()
			})
			defer sortFiles(selected, mlabel)
		}
	})

	cbutton.Connect("clicked", func(h *gtk.Button) {
		determinateFlag = true

		mainOpenedWindow.Destroy()

		initWin()
	})

	box.Add(list)
	list.SetSelectionMode(3)
	list.SelectAll()

	box.Add(button)
	box.Add(sbutton)
	box.Add(cbutton)

	if tcount == 0 {
		mlabel.SetText("Все файлы обнаружены")
	}
	box.ShowAll()
}

func sortFiles(inFiles []string, mlabel *gtk.Label) {
	if(len(inFiles)>0){
		var filesHashes []string
		err := os.Chdir(inputPath)
		if err != nil {
			log.Panic(err)
		}
		existance, err := exists(filepath.Join(inputPath, "sorted"))
		if err != nil {
			log.Panic(err)
		}
		if !existance {
			createFolderIfDontExist(true, filepath.Clean(inputPath), "sorted")
		}
		for i := 0; i < len(inFiles); i++ {
			cache, err := ioutil.ReadFile(inFiles[i])
			if err != nil {
				log.Panic(err)
			}
			hash := getHash(cache)
			if !contains(filesHashes, hash) {
				filesHashes = append(filesHashes, hash)
				tcy,month,_:=getFileDate(inFiles[i])
				createFolderIfDontExist(true, filepath.Join(inputPath,"sorted"), tcy)
				createFolderIfDontExist(true, filepath.Join(inputPath,"sorted",tcy), month)
				finalDir:=filepath.Join(inputPath,"sorted",tcy,month)
				if(filepath.Join(finalDir, filepath.Base(inFiles[i]))!=inFiles[i]){
					err = CopyFile(inFiles[i], finalDir)
					if err != nil {
						log.Printf("Копирование файла не удалось %q\n", err)
					}
				}
				err = os.Chdir(filepath.Join(inputPath, "sorted"))
				if err != nil {
					log.Panic(err)
				}else{
					if needToDelete && (filepath.Join(finalDir, filepath.Base(inFiles[i]))!=inFiles[i]){
						os.Remove(inFiles[i])
					}
					log.Println("Файл отсортирован")
				}
			}else if needToDelete{
				os.Remove(inFiles[i])
			}
		}

		err = os.Chdir(inputPath)
		if err != nil {
			log.Panic(err)
		}
		

		onDeadMessage = "Всё отсортировано. Вы найдёте файлы в папке sorted"
		
	}else{
		onDeadMessage = "Ничего не было выбрано"
	}

	determinateFlag = true

	mainOpenedWindow.Destroy()

	initWin()
}

func getFileDate(path string) (string,string,string){
	if sortingMethod{
		exifData, err := exif.SearchFileAndExtractExif(path)

		if err != nil { 
			log.Println(err)
		}else{
			tagIndex := exif.NewTagIndex()

			ifdMapping, err := exifcommon.NewIfdMappingWithStandard()
			
			if err != nil { 
				log.Println(err)
			}else{
				_, ifdIndex, err := exif.Collect(ifdMapping, tagIndex, exifData)
				
				if err != nil { 
					log.Println(err)
				}else{
					for _, ifd := range ifdIndex.Ifds {
						for _, entry := range ifd.Entries() {
							if(entry.TagName()=="DateTimeOriginal"){
								value, _ := entry.Value()
								str := fmt.Sprintf("%s", value)
								return GetTime(str)
							}
						}
					}
				}
			}
		}
	}
	tempStat, err := os.Stat(path)
	if err != nil {
		log.Panic(err)
	}
	ctime := tempStat.ModTime()
	return strconv.Itoa(ctime.Year()), ctime.Month().String(), strconv.Itoa(ctime.Day())
}

func GetTime(inp string)(string,string,string){
	tm,err:=strconv.Atoi(string([]rune(inp)[5:7]))
	if err!=nil{
		log.Panic(err)
	}
	return string([]rune(inp)[0:4]), time.Month(tm).String(), string([]rune(inp)[8:10])
}

func createFolderIfDontExist(chdir bool, path string, name string) {
	existance, err := exists(filepath.Join(path, name))
	if err != nil {
		log.Panic(err)
	}
	if !existance {
		err = os.Mkdir(name, 0770)
		if err != nil {
			log.Panic(err)
		} else {
			if chdir {
				err = os.Chdir(filepath.Join(path, name))
				if err != nil {
					log.Panic(err)
				}
			}
			return
		}
	} else {
		if chdir {
			err = os.Chdir(filepath.Join(path, name))
			if err != nil {
				log.Panic(err)
			}
		}
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (m Menu) GtkWidget() *gtk.Widget {
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)

	if err != nil {
		log.Panic(err)
	}

	box.SetMarginTop(24)
	box.SetMarginBottom(24)
	box.SetMarginStart(24)
	box.SetMarginEnd(24)

	m.rebuildWidget(box)

	return &box.Widget
}

func (m Menu) rebuildWidget(box *gtk.Box) {
	box.GetChildren().Foreach(func(child any) {
		btn, _ := child.(*gtk.Widget)
		btn.Destroy()
	})
	for i, entry := range m.entries {
		switch entry.EntryType {
		case 0:
			textFB := entry.Label
			button, err := gtk.ButtonNewWithLabel(textFB)

			if err != nil {
				log.Panic(err)
			}

			k := i
			button.Connect("clicked", func(h *gtk.Button) {
				m.entries[k].Use().rebuildWidget(box)
			})

			box.Add(button)
		case 1:
			FCB, err := gtk.FileChooserButtonNew("Выбрать", 2)

			FCB.SetCurrentFolder("./")

			FCB.Connect("file-set", func() {
				inputPath = filepath.Clean(decodeUTF8(strings.Replace(FCB.GetURI(), "file://", "", 1)))
			})

			if err != nil {
				log.Panic(err)
			}

			box.Add(FCB)
		case 2:
			text := entry.Label
			button, err := gtk.ButtonNewWithLabel(text)
			
			if err != nil {
				log.Panic(err)
			}

			k := i
			button.Connect("clicked", func(h *gtk.Button) {
				h.SetLabel(m.entries[k].Use().entries[0].Label)
				h.Show()
			})

			box.Add(button)
		default:
			log.Println("Ошибка с созданием", entry.Label)
		}
	}
	box.ShowAll()
}

func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	dst = filepath.Join(dst,sfi.Name())
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("Это не файл %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("Это не файл %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func decodeUTF8(in string) string {
	newstr, err := url.QueryUnescape(in)
	if err != nil {
		log.Panic(err)
	}

	return newstr
}

func getHash(in []byte) string {
	hasher := sha1.New()
	hasher.Write(in)
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return sha
}

func contains(where []string, what string) bool {
	for _, v := range where {
		if v == what {
			return true
			break
		}
	}
	return false
}