// Copyright (2019) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package vnc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	log "minilog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Events can be written to a vnc connect
type Event interface {
	Write(w io.Writer) error
}

// WaitForItEvent is a pseudo event indicating that we should wait for an image
// to appear on the screen.
type WaitForItEvent struct {
	File    string
	Timeout time.Duration
}

// ClickItEvent is a pseudo event indicating that we should wait for an image
// to appear on the screen and click it.
type ClickItEvent struct {
	File    string
	Timeout time.Duration
}

// LoadFileEvent is a pseudo event indicating that we should start reading
// events from a different file.
type LoadFileEvent struct {
	File string
}

const (
	keyEventFmt     = "KeyEvent,%t,%s"
	pointerEventFmt = "PointerEvent,%d,%d,%d"
)

func parseEvent(cmd string) (interface{}, error) {
	fields := strings.Split(cmd, ",")

	switch fields[0] {
	case "KeyEvent":
		if len(fields) != 3 {
			return nil, fmt.Errorf("expected 2 values for KeyEvent, got %v", len(fields)-1)
		}

		down, err := strconv.ParseBool(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid KeyEvent: %v", err)
		}

		e := &KeyEvent{}
		if down {
			e.DownFlag = 1
		}

		e.Key, err = xStringToKeysym(fields[2])
		if err != nil {
			_, err = fmt.Sscanf(fields[2], "%U", &e.Key)
			if err != nil {
				return nil, fmt.Errorf("unknown key: `%s`", fields[2])
			}
		}

		return e, nil
	case "PointerEvent":
		if len(fields) != 4 {
			return nil, fmt.Errorf("expected 3 values for PointerEvent, got %v", len(fields)-1)
		}

		mask, err := strconv.ParseUint(fields[1], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid PointerEvent: %v", err)
		}

		x, err := strconv.ParseUint(fields[2], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid PointerEvent: %v", err)
		}

		y, err := strconv.ParseUint(fields[3], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid PointerEvent: %v", err)
		}

		e := &PointerEvent{
			ButtonMask: uint8(mask),
			XPosition:  uint16(x),
			YPosition:  uint16(y),
		}

		return e, nil
	case "LoadFile":
		if len(fields) != 2 {
			return nil, fmt.Errorf("expected 1 values for LoadFile, got %v", len(fields)-1)
		}

		e := &LoadFileEvent{
			File: fields[1],
		}

		return e, nil
	case "WaitForIt":
		if len(fields) != 3 {
			return nil, fmt.Errorf("expected 2 values for WaitForIt, got %v", len(fields)-1)
		}

		timeout, err := parseDuration(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid WaitForIt: %v", err)
		}

		e := &WaitForItEvent{
			File:    fields[1],
			Timeout: timeout,
		}

		return e, nil
	case "ClickIt":
		if len(fields) != 3 {
			return nil, fmt.Errorf("expected 2 values for ClickIt, got %v", len(fields)-1)
		}

		timeout, err := parseDuration(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid ClickIt: %v", err)
		}

		e := &ClickItEvent{
			File:    fields[1],
			Timeout: timeout,
		}

		return e, nil
	}

	return nil, errors.New("invalid event specified")
}

func parseDuration(s string) (time.Duration, error) {
	// unitless integer is assumed to be in nanoseconds
	if v, err := strconv.Atoi(s); err == nil {
		return time.Duration(v) * time.Nanosecond, nil
	}

	return time.ParseDuration(s)
}

// getDuration returns the duration of a given playback file
func getDuration(f *os.File) time.Duration {
	// go back to the beginning of the file
	defer f.Seek(0, 0)

	d := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := strings.SplitN(scanner.Text(), ":", 2)
		// Ignore blank and malformed lines
		if len(s) != 2 {
			log.Debug("malformed vnc statement: %s", scanner.Text())
			continue
		}

		// Ignore comments in the vnc file
		if s[0] == "#" {
			continue
		}

		i, err := strconv.Atoi(s[0])
		if err != nil {
			log.Errorln(err)
			return 0
		}
		d += i
	}

	return time.Duration(d) * time.Nanosecond
}
