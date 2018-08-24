// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type StructView struct {
	gi.Frame
	Struct      interface{} `desc:"the struct that we are a view onto"`
	Title       string      `desc:"title / prompt to show above the editor fields"`
	FieldViews  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave     ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig     ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	ToolbarStru interface{} `desc:"the struct that we successfully set a toolbar for"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, StructViewProps)

var StructViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"#title": ki.Props{
		"max-width":      units.NewValue(-1, units.Px),
		"text-align":     gi.AlignCenter,
		"vertical-align": gi.AlignTop,
	},
}

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructView) SetStruct(st interface{}, tmpSave ValueView) {
	updt := false
	if sv.Struct != st {
		updt = sv.UpdateStart()
		sv.Struct = st
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				// todo: check for delete??
				svv, _ := recv.Embed(KiT_StructView).(*StructView)
				svv.UpdateFields()
				svv.ViewSig.Emit(svv.This, 0, nil)
			})
		}
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromStruct()
	sv.UpdateEnd(updt)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *StructView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_Frame, "struct-grid")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (sv *StructView) StdConfig() (mods, updt bool) {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := sv.StdFrameConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// SetTitle sets the optional title and updates the Title label
func (sv *StructView) SetTitle(title string) {
	sv.Title = title
	if sv.Title != "" {
		lab, _ := sv.TitleWidget()
		if lab != nil {
			lab.Text = title
		}
	}
}

// Title returns the title label widget, and its index, within frame -- nil,
// -1 if not found
func (sv *StructView) TitleWidget() (*gi.Label, int) {
	idx, ok := sv.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Label), idx
}

// StructGrid returns the grid layout widget, which contains all the fields
// and values, and its index, within frame -- nil, -1 if not found
func (sv *StructView) StructGrid() (*gi.Frame, int) {
	idx, ok := sv.Children().IndexByName("struct-grid", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Frame), idx
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	sg, _ := sv.StructGrid()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewValue(10, units.Em))
	sg.SetMinPrefWidth(units.NewValue(10, units.Em))
	sg.SetStretchMaxHeight() // for this to work, ALL layers above need it too
	sg.SetStretchMaxWidth()  // for this to work, ALL layers above need it too
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	kit.FlatFieldsValueFunc(sv.Struct, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		vv := FieldToValueView(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("label-%v", field.Name)
		valnm := fmt.Sprintf("value-%v", field.Name)
		config.Add(gi.KiT_Label, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		sv.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	for i, vv := range sv.FieldViews {
		lbl := sg.KnownChild(i * 2).(*gi.Label)
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_StructView).(*StructView)
			// note: updating here is redundant -- relevant field will have already updated
			svv.ViewSig.Emit(svv.This, 0, nil)
		})
		lbltag := vvb.Field.Tag.Get("label")
		if lbltag != "" {
			lbl.Text = lbltag
		} else {
			lbl.Text = vvb.Field.Name
		}
		lbl.Tooltip = vvb.Field.Tag.Get("desc")
		widg := sg.KnownChild((i * 2) + 1).(gi.Node2D)
		widg.SetProp("horizontal-align", gi.AlignLeft)
		vv.ConfigWidget(widg)
	}
	sg.UpdateEnd(updt)
}

// UpdateFromStruct updates full widget layout from structure
func (sv *StructView) UpdateFromStruct() {
	mods, updt := sv.StdConfig()
	sv.ConfigStructGrid()
	sv.ConfigToolbar()
	if mods {
		sv.UpdateEnd(updt)
	}
}

// UpdateFields updates each of the value-view widgets for the fields --
// called by the ViewSig update
func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

// ConfigToolbar adds a toolbar based on the methview ToolBarView function, if
// one has been defined for this struct type through its registered type
// properties.
func (sv *StructView) ConfigToolbar() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	win := sv.ParentWindow()
	if win == nil {
		return // not ready yet
	}
	if sv.ToolbarStru == sv.Struct {
		return
	}
	if HasToolBarView(sv.Struct) {
		tb := &gi.ToolBar{}
		tb.InitName(tb, "structview-tbar")
		ok := ToolBarView(sv.Struct, win, tb)
		if ok {
			_, idx := sv.StructGrid()
			sv.InsertChild(tb, idx)
			sv.ToolbarStru = sv.Struct
			return
		}
		sv.ToolbarStru = sv.Struct // prevent repeat error messages
	}
	sv.DeleteToolbar() // delete any old one
}

// DeleteToolbar deletes any existing toolbar
func (sv *StructView) DeleteToolbar() {
	_, idx := sv.StructGrid()
	if _, ok := sv.KnownChild(idx - 1).(*gi.ToolBar); ok {
		sv.DeleteChildAtIndex(idx-1, true)
	}
	sv.ToolbarStru = nil
}
