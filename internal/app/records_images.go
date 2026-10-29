package app

import (
	"bytes"
	"errors"
	"image"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
	"trajon/internal/journal"
)

var errRecordImageLimit = errors.New("too many journal screenshots")

func (a *application) recordImageJobCurrent(job uint64, recordID string) bool {
	s := &a.records
	return !s.closed && s.ready && s.review && s.book.Active().CanReview() && s.imageJob == job && s.book.ActiveID == recordID
}

func (a *application) chooseRecordImages() {
	s, win := &a.records, a.recordsEditorWindow()
	if !s.ready || !s.review || !s.book.Active().CanReview() || s.imageBusy || win == nil {
		return
	}
	s.closeAfterSave = false
	s.imageJob++
	job, id := s.imageJob, s.book.ActiveID
	s.imageBusy, s.imageError = true, nil
	title := a.text().JournalAddScreenshot()
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: title, Multiple: true,
			Filters: []mygo.FileFilter{{Name: title, Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "bmp"}}}})
		a.updateRecords(win, func() {
			if !a.recordImageJobCurrent(job, id) {
				return
			}
			s.imageBusy = false
			if err != nil {
				s.imageError = err
				return
			}
			if len(paths) > 0 {
				a.importRecordImages(paths, nil)
			}
		})
	}()
}

// The caller supplies a materialized native transfer, never a frame-local UI
// element. Only Review builds drop targets and image-aware paste handlers.
func (a *application) pasteRecordImages(services ui.Services) bool {
	if !a.records.ready || !a.records.review || !a.records.book.Active().CanReview() {
		return false
	}
	formats := services.ClipboardFormats()
	if !slices.Contains(formats, transfer.PNG) && !slices.Contains(formats, transfer.FileList) {
		return false
	}
	data := services.ReadClipboardData(transfer.PNG, transfer.FileList)
	a.acceptRecordImageData(data)
	return true
}

func (a *application) acceptRecordImageData(data transfer.Data) {
	if !a.records.ready || !a.records.review || !a.records.book.Active().CanReview() {
		return
	}
	if paths, err := data.Files(); err == nil && len(paths) > 0 {
		a.importRecordImages(paths, nil)
		return
	}
	if png, err := data.Read(transfer.PNG); err == nil {
		a.importRecordImages(nil, png)
	}
}

func (a *application) importRecordImages(paths []string, raw []byte) {
	s, win := &a.records, a.recordsEditorWindow()
	if !s.ready || !s.review || !s.book.Active().CanReview() || s.imageBusy || s.closed || win == nil {
		return
	}
	remaining := journal.MaxScreenshots - len(s.book.Active().Screenshots)
	if remaining <= 0 {
		s.imageError = errRecordImageLimit
		return
	}
	s.closeAfterSave = false
	s.imageJob++
	job, id, dir := s.imageJob, s.book.ActiveID, s.dir
	s.imageBusy, s.imageError = true, nil
	paths, raw = slices.Clone(paths), bytes.Clone(raw)
	seen := map[string]bool{}
	for _, screenshot := range s.book.Active().Screenshots {
		seen[screenshot.File] = true
	}
	go func() {
		type imported struct {
			screenshot journal.Screenshot
			bitmap     *ui.Bitmap
		}
		var images []imported
		var importErr error
		add := func(name string, data []byte) {
			if len(images) >= remaining {
				importErr = errRecordImageLimit
				return
			}
			screenshot, normalized, err := journal.ImportImage(dir, name, data)
			if err != nil {
				importErr = err
				return
			}
			if seen[screenshot.File] {
				return
			}
			seen[screenshot.File] = true
			bitmap, err := decodeRecordImage(normalized)
			if err != nil {
				importErr = err
				return
			}
			images = append(images, imported{screenshot, bitmap})
		}
		if len(raw) > 0 {
			add("screenshot.png", raw)
		}
		for _, path := range paths {
			if len(images) >= remaining {
				importErr = errRecordImageLimit
				break
			}
			data, err := journal.ReadImageFile(path)
			if err != nil {
				importErr = err
				continue
			}
			add(path, data)
		}
		a.updateRecords(win, func() {
			if !a.recordImageJobCurrent(job, id) {
				return
			}
			s.imageBusy, s.imageError = false, importErr
			for _, result := range images {
				s.book.Active().AddScreenshot(result.screenshot)
				if s.images == nil {
					s.images = map[string]*ui.Bitmap{}
				}
				s.images[result.screenshot.File] = result.bitmap
			}
			if len(images) > 0 {
				a.changeRecord()
			}
		})
	}()
}

func decodeRecordImage(data []byte) (*ui.Bitmap, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > journal.MaxImagePixels {
		return nil, journal.ErrImage
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	w, h := config.Width, config.Height
	if max(w, h) > 1600 {
		scale := 1600.0 / float64(max(w, h))
		small := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))))
		draw.CatmullRom.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)
		img = small
	}
	return ui.NewBitmap(img), nil
}

func (a *application) loadRecordImages() {
	s, win := &a.records, a.recordsEditorWindow()
	s.images, s.imageErrors = map[string]*ui.Bitmap{}, map[string]error{}
	if !s.ready || win == nil {
		return
	}
	id, dir, job := s.book.ActiveID, s.dir, s.imageJob
	files := slices.Clone(s.book.Active().Screenshots)
	go func() {
		for _, screenshot := range files {
			data, err := journal.LoadImage(dir, screenshot.File)
			var bitmap *ui.Bitmap
			if err == nil {
				bitmap, err = decodeRecordImage(data)
			}
			a.updateRecords(win, func() {
				if s.closed || s.imageJob != job || s.book.ActiveID != id || !slices.ContainsFunc(s.book.Active().Screenshots, func(i journal.Screenshot) bool { return i.File == screenshot.File }) {
					return
				}
				s.images[screenshot.File], s.imageErrors[screenshot.File] = bitmap, err
			})
		}
	}()
}
