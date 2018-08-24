// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/fatih/camelcase"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// MethViewErr is error logging function for MethView system, showing the type info
func MethViewErr(vtyp reflect.Type, msg string) {
	if vtyp != nil {
		log.Printf("giv.MethodView for type: %v: debug error: %v\n", vtyp.String(), msg)
	} else {
		log.Printf("giv.MethodView debug error: %v\n", msg)
	}
}

// MethViewTypeProps gets props, typ of val, returns false if not found or
// other err
func MethViewTypeProps(val interface{}) (ki.Props, reflect.Type, bool) {
	if kit.IfaceIsNil(val) {
		return nil, nil, false
	}
	vtyp := reflect.TypeOf(val)
	tpp := kit.Types.Properties(kit.NonPtrType(vtyp), false)
	if tpp == nil {
		return tpp, vtyp, false
	}
	return tpp, vtyp, true
}

// HasMainMenuView returns true if given val has a MainMenu type property
// registered -- call this to check before then calling MainMenuView
func HasMainMenuView(val interface{}) bool {
	tpp, _, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceProps(tpp, "MainMenu")
	if !ok {
		return false
	}
	return true
}

// MainMenuView configures the given MenuBar according to the "MainMenu"
// properties registered on the type for given value element, through the
// kit.AddType method.  See https://github.com/goki/gi/wiki/Views for full
// details on formats and options for configuring the menu.  Returns false on
// errors.
func MainMenuView(val interface{}, win *gi.Window, mbar *gi.MenuBar) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	mp, ok := ki.SliceProps(tpp, "MainMenu")
	if !ok {
		return false
	}

	mnms := make([]string, len(mp))
	for mmi, mm := range mp {
		if mm.Name == "AppMenu" {
			mnms[mmi] = oswin.TheApp.Name()
		} else {
			mnms[mmi] = mm.Name
		}
	}
	mbar.ConfigMenus(mnms)
	rval := true
	for mmi, mm := range mp {
		ma := mbar.KnownChild(mmi).(*gi.Action)
		if mm.Name == "AppMenu" {
			ma.Menu.AddAppMenu(win)
			continue
		}
		if mm.Name == "Edit" {
			if ms, ok := mm.Value.(string); ok {
				if ms == "Copy Cut Paste" {
					ma.Menu.AddCopyCutPaste(win, false)
				} else {
					MethViewErr(vtyp, fmt.Sprintf("Unrecognized Edit menu special string: %v -- `Copy Cut Paste` is standard", ms))
				}
			}
		}
		rv := ActionsView(val, vtyp, win, ma, mm.Value)
		if !rv {
			rval = false
		}
	}
	win.MainMenuUpdated()
	return rval
}

// HasToolBarView returns true if given val has a ToolBar type property
// registered -- call this to check before then calling ToolBarView.
func HasToolBarView(val interface{}) bool {
	tpp, _, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceProps(tpp, "ToolBar")
	if !ok {
		return false
	}
	return true
}

// ToolBarView configures ToolBar according to the "ToolBar" properties
// registered on the type for given value element, through the kit.AddType
// method.  See https://github.com/goki/gi/wiki/Views for full details on
// formats and options for configuring the menu.  Returns false on errors.
func ToolBarView(val interface{}, win *gi.Window, tb *gi.ToolBar) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	tp, ok := ki.SliceProps(tpp, "ToolBar")
	if !ok {
		return false
	}

	rval := true
	for _, te := range tp {
		if strings.HasPrefix(te.Name, "sep-") {
			sep := tb.AddNewChild(gi.KiT_Separator, te.Name).(*gi.Separator)
			sep.Horiz = false
			continue
		}
		ac := tb.AddNewChild(gi.KiT_Action, te.Name).(*gi.Action)
		rv := ActionsView(val, vtyp, win, ac, te.Value)
		if !rv {
			rval = false
		}
	}
	return rval
}

// ActionsView processes properties for parent action pa for overall object
// val of given type -- could have a sub-menu of further actions or might just
// be a single action
func ActionsView(val interface{}, vtyp reflect.Type, win *gi.Window, pa *gi.Action, pp interface{}) bool {
	pa.Text = strings.Replace(strings.Join(camelcase.Split(pa.Nm), " "), "  ", " ", -1)
	rval := true
	switch pv := pp.(type) {
	case ki.PropSlice:
		for _, mm := range pv {
			if strings.HasPrefix(mm.Name, "sep-") {
				pa.Menu.AddSeparator(mm.Name)
			} else {
				nac := &gi.Action{}
				nac.InitName(nac, mm.Name)
				nac.SetAsMenu()
				pa.Menu = append(pa.Menu, nac.This.(gi.Node2D))
				rv := ActionsView(val, vtyp, win, nac, mm.Value)
				if !rv {
					rval = false
				}
			}
		}
	case ki.BlankProp:
		rv := ActionView(val, vtyp, win, pa, nil)
		if !rv {
			rval = false
		}
	case ki.Props:
		rv := ActionView(val, vtyp, win, pa, pv)
		if !rv {
			rval = false
		}
	}
	return rval
}

// ActionView configures given action with given props
func ActionView(val interface{}, vtyp reflect.Type, win *gi.Window, ac *gi.Action, props ki.Props) bool {
	// special action names
	switch ac.Nm {
	case "Close Window":
		ac.Shortcut = "Command+W"
		ac.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			win.OSWin.CloseReq()
		})
		return true
	}

	methNm := ac.Nm
	methTyp, hasmeth := vtyp.MethodByName(methNm)
	if !hasmeth {
		MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- not found in type", methNm))
		return false
	}
	valval := reflect.ValueOf(val)
	methVal := valval.MethodByName(methNm)
	if kit.ValueIsZero(methVal) || methVal.IsNil() {
		MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- method value not valid", methNm))
		return false
	}

	rval := true
	md := &MethViewData{Val: val, ValVal: valval, Win: win, Method: methNm, MethVal: methVal, MethTyp: methTyp}
	ac.Data = md
	if props == nil {
		ac.ActionSig.Connect(win.This, MethViewCall)
		return true
	}
	fv, hasfv := props["FileView"]
	// sv, hassv := props["SliceViewSelect"]
	for pk, pv := range props {
		switch pk {
		case "FileView":
		case "SliceViewSelect":
		case "shortcut":
			ac.Shortcut = kit.ToString(pv)
		case "label":
			ac.Text = kit.ToString(pv)
		case "desc":
			md.Desc = kit.ToString(pv)
		case "confirm":
			bitflag.Set32((*int32)(&(md.Flags)), int(MethViewConfirm))
		case "update-after":
			bitflag.Set32((*int32)(&(md.Flags)), int(MethViewUpdateAfter))
		case "full-update-after":
			bitflag.Set32((*int32)(&(md.Flags)), int(MethViewFullUpdateAfter))
		case "show-return":
			bitflag.Set32((*int32)(&(md.Flags)), int(MethViewShowReturn))
		case "Args":
			argv, ok := pv.(ki.PropSlice)
			if !ok {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, Args property must be of type ki.PropSlice, containing names and other properties for each arg", methNm))
				rval = false
			} else {
				if ActionViewArgsValidate(vtyp, methTyp, argv) {
					md.ArgProps = argv
				} else {
					rval = false
				}
			}
		}
	}
	if !rval {
		return false
	}
	if hasfv {
		fvp, ok := fv.(ki.Props)
		if !ok {
			MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, FileView property must be of type ki.Props, containing params for the FileView dialog", methNm))
			rval = false
		} else {
			md.SpecProps = fvp
			ac.ActionSig.Connect(win.This, MethFileView)
		}
		return rval
	}
	ac.ActionSig.Connect(win.This, MethViewCall)
	return true
}

// ActionViewArgsValidate validates the Args properties relative to number of args on type
func ActionViewArgsValidate(vtyp reflect.Type, meth reflect.Method, argprops ki.PropSlice) bool {
	mtyp := meth.Type
	narg := mtyp.NumIn()
	apsz := len(argprops)
	if narg-1 != apsz {
		MethViewErr(vtyp, fmt.Sprintf("Method: %v takes %v args (beyond the receiver), but Args properties only has %v", meth.Name, narg-1, apsz))
		return false
	}
	return true
}

//////////////////////////////////////////////////////////////////////////////////
//    Method Callbacks -- called when Action fires

// MethViewFlags define bitflags for method view action options
type MethViewFlags int32

const (
	// MethViewConfirm confirms action before proceeding
	MethViewConfirm MethViewFlags = iota

	// MethViewUpdateAfter does render after method is called
	MethViewUpdateAfter

	// MethViewFullUpdateAfter does full re-render after method is called
	MethViewFullUpdateAfter

	// MethViewShowReturn shows the return value from the method
	MethViewShowReturn

	MethViewFlagsN
)

//go:generate stringer -type=MethViewFlags

var KiT_MethViewFlags = kit.Enums.AddEnumAltLower(MethViewFlagsN, true, nil, "MethView") // true = bitflags

// MethViewData is set to the Action.Data field for all MethView actions,
// containing info needed to actually call the Method on value Val.
type MethViewData struct {
	Val       interface{}
	ValVal    reflect.Value
	Win       *gi.Window
	Method    string
	MethVal   reflect.Value
	MethTyp   reflect.Method
	ArgProps  ki.PropSlice `desc:"names and other properties of args, in one-to-one with method args"`
	SpecProps ki.Props     `desc:"props for special action types, e.g., FileView"`
	Desc      string
	Flags     MethViewFlags
}

// MethViewCall is the receiver func for MethView actions that call a method
// -- it uses the MethViewData to call the target method.
func MethViewCall(recv, send ki.Ki, sig int64, data interface{}) {
	ac := send.(*gi.Action)
	md := ac.Data.(*MethViewData)
	if md.ArgProps == nil { // no args -- just call
		if bitflag.Has32(int32(md.Flags), int(MethViewConfirm)) {
			gi.PromptDialog(md.Win.Viewport, ac.Text, md.Desc, true, true, nil, md.Win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					MethViewCallMeth(md, nil)
				}
			})
		} else {
			MethViewCallMeth(md, nil)
		}
		return
	}
	// need to prompt for args
	ad, args, ok := MethViewArgData(md)
	if !ok {
		return
	}
	ArgViewDialog(md.Win.Viewport, ad, ac.Text, md.Desc, nil, md.Win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.DialogAccepted) {
			MethViewCallMeth(md, args)
		}
	})
}

// MethViewCallMeth calls the method with given args, and processes the
// results as specified in the MethViewData.
func MethViewCallMeth(md *MethViewData, args []reflect.Value) {
	rv := md.MethVal.Call(args)
	if bitflag.Has32(int32(md.Flags), int(MethViewUpdateAfter)) {
		// todo: only for ki types, or else do window I guess..
	}
	if bitflag.Has32(int32(md.Flags), int(MethViewFullUpdateAfter)) {
		// todo: only for ki types
	}
	if bitflag.Has32(int32(md.Flags), int(MethViewShowReturn)) {
		gi.PromptDialog(md.Win.Viewport, md.Method+" Result", rv[0].String(), true, false, nil, nil, nil)
	}
}

// ArgData contains the relevant data for each arg, including the
// reflect.Value, name, optional description, and default value
type ArgData struct {
	Val     reflect.Value
	Name    string
	Desc    string
	Default interface{}
}

// MethViewArgData gets the arg data for the method args, returns false if errors
func MethViewArgData(md *MethViewData) ([]ArgData, []reflect.Value, bool) {
	mtyp := md.MethTyp.Type
	narg := mtyp.NumIn() - 1
	ads := make([]ArgData, narg)
	args := make([]reflect.Value, narg)
	for ai := 0; ai < narg; ai++ {
		ad := &ads[ai]
		atyp := mtyp.In(1 + ai)
		av := reflect.New(atyp)
		ad.Val = av
		args[ai] = av.Elem()

		aps := &md.ArgProps[ai]
		ad.Name = aps.Name
		switch apv := aps.Value.(type) {
		case ki.BlankProp:
		case ki.Props:
			// fv, hasfv := apv["FileView"]
			// sv, hassv := props["SliceViewSelect"]
			for pk, pv := range apv {
				switch pk {
				case "FileView":
				case "SliceViewSelect":
				case "desc":
					ad.Desc = kit.ToString(pv)
				case "default":
					ad.Default = pv
				case "default-field":
					field := pv.(string)
					if flv, ok := MethViewFieldValue(md.ValVal, field); ok {
						ad.Default = flv.Interface()
					}
				}
			}
		}
	}
	return ads, args, true
}

// MethFileView is the receiver func for MethView actions that open the
// FileView dialog prior to calling the method.
func MethFileView(recv, send ki.Ki, sig int64, data interface{}) {
	ac := send.(*gi.Action)
	md := ac.Data.(*MethViewData)
	meth := md.ValVal.MethodByName(md.Method)
	if kit.ValueIsZero(meth) || meth.IsNil() {
		log.Printf("giv.MethViewCall: Method %v not found in type: %v\n", md.Method, md.ValVal.Type().String())
		return
	}

	ext := ""
	field := ""
	def := ""
	if ep, ok := md.SpecProps["ext"]; ok {
		ext = ep.(string)
	}
	if fp, ok := md.SpecProps["field"]; ok {
		field = fp.(string)
	}
	if dp, ok := md.SpecProps["default"]; ok {
		def = dp.(string)
	}

	var fval *reflect.Value
	if field != "" {
		if flv, ok := MethViewFieldValue(md.ValVal, field); ok {
			fval = flv
			def = kit.ToString(fval.Interface())
		}
	}

	FileViewDialog(md.Win.Viewport, def, ext, ac.Text, "", nil, nil, md.Win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.DialogAccepted) {
			dlg, _ := send.(*gi.Dialog)
			args := make([]reflect.Value, 1)
			fn := FileViewDialogValue(dlg)
			if fval != nil {
				kit.SetRobust(kit.PtrValue(*fval).Interface(), fn)
			}
			args[0] = reflect.ValueOf(fn)
			MethViewCallMeth(md, args)
		}
	})
}

// MethViewFieldValue returns a reflect.Value for the given field name,
// checking safely (false if not found)
func MethViewFieldValue(vval reflect.Value, field string) (*reflect.Value, bool) {
	typ := kit.NonPtrType(vval.Type())
	_, ok := typ.FieldByName(field)
	if !ok {
		log.Printf("giv.MethViewFieldValue: Could not find field %v in type: %v\n", field, typ.String())
		return nil, false
	}
	fv := kit.NonPtrValue(vval).FieldByName(field)
	return &fv, true
}
