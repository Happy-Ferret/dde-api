/**
 * Copyright (c) 2011 ~ 2013 Deepin, Inc.
 *               2011 ~ 2013 jouyouyun
 *
 * Author:      jouyouyun <jouyouwen717@gmail.com>
 * Maintainer:  jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, see <http://www.gnu.org/licenses/>.
 **/

package main

// #cgo CFLAGS: -g -Wall
// #cgo pkg-config: x11 xtst glib-2.0
// #include "record.h"
import "C"

import (
	"dlib"
	"dlib/dbus"
	dlogger "dlib/logger"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
	"os"
	"sync"
)

const (
	MotionFlag = int32(1 << 0)
	ButtonFlag = int32(1 << 1)
	KeyFlag    = int32(1 << 2)
	AllFlag    = int32(MotionFlag | ButtonFlag | KeyFlag)
)

var (
	logger     = dlogger.NewLogger("dde-api/mousearea")
	X          *xgbutil.XUtil
	opMouse    *Manager
	idRangeMap map[int32]*coordinateInfo
)

var genID = func() func() int32 {
	var lock sync.Mutex
	id := int32(0)
	return func() int32 {
		lock.Lock()
		tmp := id
		id += 1
		lock.Unlock()
		return tmp
	}
}()

type coordinateInfo struct {
	areas        []coordinateRange
	moveIntoFlag bool
	motionFlag   bool
	buttonFlag   bool
	keyFlag      bool
}

func getEventFlags(flag int32) (bool, bool, bool) {
	motionFlag := false
	buttonFlag := false
	keyFlag := false

	if flag >= 0 && flag <= int32(AllFlag) {
		if (flag & MotionFlag) == MotionFlag {
			motionFlag = true
		}

		if (flag & ButtonFlag) == ButtonFlag {
			buttonFlag = true
		}

		if (flag & KeyFlag) == KeyFlag {
			keyFlag = true
		}
	}

	return motionFlag, buttonFlag, keyFlag
}

func (op *Manager) RegisterArea(x1, y1, x2, y2, flag int32) int32 {
	cookie := genID()
	logger.Info("ID: ", cookie)

	info := &coordinateInfo{}
	info.areas = []coordinateRange{coordinateRange{x1, y1, x2, y2}}
	info.moveIntoFlag = false
	info.motionFlag, info.buttonFlag, info.keyFlag = getEventFlags(flag)
	idRangeMap[cookie] = info

	return cookie
}

func (op *Manager) RegisterAreas(area []coordinateRange, flag int32) int32 {
	cookie := genID()
	logger.Info("ID: ", cookie)

	info := &coordinateInfo{}
	info.areas = area
	info.moveIntoFlag = false
	info.motionFlag, info.buttonFlag, info.keyFlag = getEventFlags(flag)
	idRangeMap[cookie] = info

	return cookie
}

func (op *Manager) RegisterFullScreen() int32 {
	if op.FullScreenId == -1 {
		cookie := genID()
		op.FullScreenId = cookie
	}
	logger.Info("ID: ", op.FullScreenId)

	return op.FullScreenId
}

func (op *Manager) UnregisterArea(cookie int32) {
	if _, ok := idRangeMap[cookie]; ok {
		delete(idRangeMap, cookie)
	}
	if cookie == op.FullScreenId {
		op.FullScreenId = -1
	}
}

func NewManager() *Manager {
	m := &Manager{}
	m.FullScreenId = -1
	return m
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {
	defer logger.EndTracing()

	if !dlib.UniqueOnSession(MOUSE_AREA_DEST) {
		logger.Warning("There already has an XMouseArea daemon running.")
		return
	}

	// configure logger
	logger.SetRestartCommand("/usr/lib/deepin-api/mousearea", "--debug")
	if stringInSlice("-d", os.Args) || stringInSlice("--debug", os.Args) {
		logger.SetLogLevel(dlogger.LEVEL_DEBUG)
	}

	var err error
	X, err = xgbutil.NewConn()
	if err != nil {
		logger.Warning("New XGB Connection Failed")
		return
	}
	keybind.Initialize(X)

	idRangeMap = make(map[int32]*coordinateInfo)
	opMouse = NewManager()
	err = dbus.InstallOnSession(opMouse)
	if err != nil {
		logger.Error("Install DBus Session Failed:", err)
		panic(err)
	}

	C.record_init()
	defer C.record_finalize()

	dbus.DealWithUnhandledMessage()
	//select {}
	if err = dbus.Wait(); err != nil {
		logger.Error("lost dbus session:", err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
