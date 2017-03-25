// Copyright 2013 Michael Yang. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package id3

import (
	"bytes"
	v2 "github.com/jcs/id3-go/v2"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

const (
	testFile        = "test.mp3"
	chapterTestFile = "chaptest.mp3"
)

func TestOpen(t *testing.T) {
	file, err := Open(testFile)
	if err != nil {
		t.Errorf("Open: unable to open file")
	}

	tag, ok := file.Tagger.(*v2.Tag)
	if !ok {
		t.Errorf("Open: incorrect tagger type")
	}

	if s := tag.Artist(); s != "Paloalto\x00" {
		t.Errorf("Open: incorrect artist, %v", s)
	}

	if s := tag.Title(); s != "Nice Life (Feat. Basick)" {
		t.Errorf("Open: incorrect title, %v", s)
	}

	if s := tag.Album(); s != "Chief Life" {
		t.Errorf("Open: incorrect album, %v", s)
	}

	parsedFrame := file.Frame("COMM")
	resultFrame, ok := parsedFrame.(*v2.UnsynchTextFrame)
	if !ok {
		t.Error("Couldn't cast frame")
	}

	expected := "✓"
	actual := resultFrame.Description()

	if expected != actual {
		t.Errorf("Expected %x, got %x", expected, actual)
	}

	actual = resultFrame.Text()
	if expected != actual {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

func TestClose(t *testing.T) {
	before, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Errorf("test file error")
	}

	file, err := Open(testFile)
	if err != nil {
		t.Errorf("Close: unable to open file")
	}
	beforeCutoff := file.originalSize

	file.SetArtist("Paloalto")
	file.SetTitle("Test test test test test test")

	afterCutoff := file.Size()

	if err := file.Close(); err != nil {
		t.Errorf("Close: unable to close file")
	}

	after, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Errorf("Close: unable to reopen file")
	}

	if !bytes.Equal(before[beforeCutoff:], after[afterCutoff:]) {
		t.Errorf("Close: nontag data lost on close")
	}

	if err := ioutil.WriteFile(testFile, before, 0666); err != nil {
		t.Errorf("Close: unable to write original contents to test file")
	}
}

func TestReadonly(t *testing.T) {
	before, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Errorf("test file error")
	}

	file, err := Open(testFile)
	if err != nil {
		t.Errorf("Readonly: unable to open file")
	}

	file.Title()
	file.Artist()
	file.Album()
	file.Year()
	file.Genre()
	file.Comments()

	if err := file.Close(); err != nil {
		t.Errorf("Readonly: unable to close file")
	}

	after, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Errorf("Readonly: unable to reopen file")
	}

	if !bytes.Equal(before, after) {
		t.Errorf("Readonly: tag data modified without set")
	}
}

func TestAddTag(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "notag")
	if err != nil {
		t.Fatal(err)
	}

	file, err := Open(tempFile.Name())
	if err != nil {
		t.Errorf("AddTag: unable to open empty file")
	}

	tag := file.Tagger

	if tag == nil {
		t.Errorf("AddTag: no tag added to file")
	}

	file.SetArtist("Michael")

	err = file.Close()
	if err != nil {
		t.Errorf("AddTag: error closing new file")
	}

	reopenBytes, err := ioutil.ReadFile(tempFile.Name())
	if err != nil {
		t.Errorf("AddTag: error reopening file")
	}

	expectedBytes := tag.Bytes()
	if !bytes.Equal(expectedBytes, reopenBytes) {
		t.Errorf("AddTag: tag not written correctly: %v", reopenBytes)
	}
}

func TestUnsynchTextFrame_RoundTrip(t *testing.T) {
	var (
		err              error
		tempfile         *os.File
		f                *File
		tagger           *v2.Tag
		ft               v2.FrameType
		utextFrame       *v2.UnsynchTextFrame
		parsedFrame      v2.Framer
		resultFrame      *v2.UnsynchTextFrame
		ok               bool
		expected, actual string
	)

	tempfile, err = ioutil.TempFile("", "id3v2")
	if err != nil {
		t.Fatal(err)
	}

	tagger = v2.NewTag(3)
	ft = v2.V23FrameTypeMap["COMM"]
	utextFrame = v2.NewUnsynchTextFrame(ft, "Comment", "Foo")
	tagger.AddFrames(utextFrame)

	_, err = tempfile.Write(tagger.Bytes())
	tempfile.Close()
	if err != nil {
		t.Fatal(err)
	}

	f, err = Open(tempfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	parsedFrame = f.Frame("COMM")
	if resultFrame, ok = parsedFrame.(*v2.UnsynchTextFrame); !ok {
		t.Error("Couldn't cast frame")
	} else {
		expected = utextFrame.Description()
		actual = resultFrame.Description()
		if expected != actual {
			t.Errorf("Expected %q, got %q", expected, actual)
		}
	}
}

func TestUTF16CommPanic(t *testing.T) {
	osFile, err := os.Open(testFile)
	if err != nil {
		t.Error(err)
	}
	tempfile, err := ioutil.TempFile("", "utf16_comm")
	if err != nil {
		t.Error(err)
	}
	io.Copy(tempfile, osFile)
	osFile.Close()
	tempfile.Close()
	for i := 0; i < 2; i++ {
		file, err := Open(tempfile.Name())
		if err != nil {
			t.Error(err)
		}
		file.Close()
	}
}

func TestTOC(t *testing.T) {
	file, err := Open(chapterTestFile)
	if err != nil {
		t.Errorf("unable to open chapter file")
	}

	_toc := file.Frame("CTOC")
	if _toc == nil {
		t.Errorf("failed finding CTOC")
	}

	toc := _toc.(*v2.TOCFrame)

	if toc.Element != "toc" {
		t.Errorf("toc element %q", toc.Element)
	}

	if !toc.TopLevel {
		t.Errorf("not top level")
	}

	if !toc.Ordered {
		t.Errorf("not ordered")
	}

	if toc.ChildElements[0] != "chp0" {
		t.Errorf("got %q", toc.ChildElements[0])
	}

	if toc.ChildElements[len(toc.ChildElements)-1] != "chp8" {
		t.Errorf("got %q", toc.ChildElements[len(toc.ChildElements)-1])
	}

	file.Close()
}

func TestChapters(t *testing.T) {
	file, err := Open(chapterTestFile)
	if err != nil {
		t.Errorf("unable to open chapter file")
	}

	_chaps := file.Frames("CHAP")
	if _chaps == nil {
		t.Errorf("failed finding CHAP")
	}

	var chaps [](*v2.ChapterFrame)
	for _, c := range _chaps {
		cc := c.(*v2.ChapterFrame)
		chaps = append(chaps, cc)
	}

	if chaps[0].Title() != "Intro" {
		t.Errorf("got %q", chaps[0].Title)
	}
	if chaps[8].Title() != "Get a free account!" {
		t.Errorf("got %q", chaps[8].Title)
	}

	if !chaps[0].UseTime {
		t.Errorf("should be using time")
	}

	if chaps[0].EndTime != 15000 {
		t.Errorf("end time is %q", chaps[0].EndTime)
	}

	file.Close()
}
