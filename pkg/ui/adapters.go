// Adapters to convert plan/state into Table rows for Bubbletea UI
package ui

import (
	"reflect"
	"strings"
)

// PlanToRows converts a plan (with Items field) to table rows. Skips InSync for human output.

func PlanToRows(plan interface{}) []Row {
	rows := []Row{}

	// Try direct assertion (for legacy code/tests)
	// Prefer newer field name "Kind" but accept legacy "Type" in older call sites.
	type planItem struct {
		Action      string
		Name        string
		Kind        string
		Region      string
		Explanation string
		Details     string
	}
	type planWrap struct{ Items []planItem }
	if typed, ok := plan.(*planWrap); ok {
		for _, it := range typed.Items {
			icon, iconStyle := StateIcon(it.Action)
			info := it.Explanation
			if info == "" {
				info = it.Details
			}
			// Build composite ID: Kind[/Region]/Name
			parts := []string{}
			if it.Kind != "" {
				parts = append(parts, it.Kind)
			}
			if it.Region != "" {
				parts = append(parts, it.Region)
			}
			if it.Name != "" {
				parts = append(parts, it.Name)
			}
			id := strings.Join(parts, "/")
			row := Row{
				Cell{Text: icon, Style: iconStyle},
				Cell{Text: id},
				Cell{Text: it.Kind},
				Cell{Text: it.Region},
				Cell{Text: it.Name},
				Cell{Text: info, Style: &InfoStyle},
			}
			rows = append(rows, row)
		}
		return rows
	}

	// Use reflection for flexible field handling
	v := reflect.ValueOf(plan)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	itemsField := v.FieldByName("Items")
	if !itemsField.IsValid() || itemsField.Kind() != reflect.Slice {
		return rows // no items
	}
	for i := 0; i < itemsField.Len(); i++ {
		it := itemsField.Index(i)
		// Extract by field name
		icon, iconStyle := StateIcon(fieldString(it, "Action"))
		name := fieldString(it, "Name")
		// Use canonical field name "Kind"
		typ := fieldString(it, "Kind")
		_ = fieldString(it, "Region") // legacy field; ignore for current table layout
		info := fieldString(it, "Explanation")
		if info == "" {
			info = fieldString(it, "Details")
		}
		// Emit columns: state, composite id (type/region/name), kind, group, name, info
		parts := []string{}
		if typ != "" {
			parts = append(parts, typ)
		}
		// region may be empty
		region := fieldString(it, "Region")
		if region != "" {
			parts = append(parts, region)
		}
		if name != "" {
			parts = append(parts, name)
		}
		id := strings.Join(parts, "/")
		row := Row{
			Cell{Text: icon, Style: iconStyle},
			Cell{Text: id},
			Cell{Text: typ},
			Cell{Text: region},
			Cell{Text: name},
			Cell{Text: info, Style: &InfoStyle},
		}
		rows = append(rows, row)
	}
	return rows
}

func fieldString(v reflect.Value, field string) string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	f := v.FieldByName(field)
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}

// StateToRows converts a state (with Items) to table rows (shows everything)
func StateToRows(state interface{}) []Row {
	rows := []Row{}
	v := reflect.ValueOf(state)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	resourcesField := v.FieldByName("Resources")
	if !resourcesField.IsValid() || resourcesField.Kind() != reflect.Slice {
		return rows
	}
	for i := 0; i < resourcesField.Len(); i++ {
		res := resourcesField.Index(i)
		kind := fieldString(res, "Kind")
		group := fieldString(res, "Group")
		itemsField := res.FieldByName("Items")
		if !itemsField.IsValid() || itemsField.Kind() != reflect.Slice {
			continue
		}
		for j := 0; j < itemsField.Len(); j++ {
			it := itemsField.Index(j)
			status := fieldString(it, "Status")
			if status == "" {
				status = "managed"
			}
			icon, iconStyle := StateIcon(status)
			name := fieldString(it, "Name")
			info := fieldString(it, "Version")

			// Build composite ID: Kind/Group/Name
			parts := []string{}
			if kind != "" {
				parts = append(parts, kind)
			}
			if group != "" {
				parts = append(parts, group)
			}
			if name != "" {
				parts = append(parts, name)
			}
			id := strings.Join(parts, "/")

			row := Row{
				Cell{Text: icon, Style: iconStyle},
				Cell{Text: id},
				Cell{Text: kind},
				Cell{Text: group},
				Cell{Text: name},
				Cell{Text: info, Style: &InfoStyle},
			}
			rows = append(rows, row)
		}
	}
	return rows
}
