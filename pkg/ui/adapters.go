// Adapters to convert plan/state into Table rows for Bubbletea UI
package ui

import (
	"reflect"
)

// PlanToRows converts a plan (with Items field) to table rows. Skips InSync for human output.

func PlanToRows(plan interface{}) []Row {
	rows := []Row{}

	// Try direct assertion (for legacy code/tests)
	type planItem struct {
		Action      string
		Name        string
		Type        string
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
			row := Row{
				Cell{Text: icon, Style: iconStyle},
				Cell{Text: it.Name},
				Cell{Text: it.Type},
				Cell{Text: it.Region},
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
		typ := fieldString(it, "Type")
		_ = fieldString(it, "Region") // legacy field; ignore for current table layout
		info := fieldString(it, "Explanation")
		if info == "" {
			info = fieldString(it, "Details")
		}
		// Emit columns: state, id/name, type, info
		row := Row{
			Cell{Text: icon, Style: iconStyle},
			Cell{Text: name},
			Cell{Text: typ},
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
	stateContainer, ok := state.(struct {
		Items []struct {
			Status string
			Name   string
			Type   string
			Region string
			Info   string
		}
	})
	if !ok {
		return rows // Can't convert
	}
	for _, it := range stateContainer.Items {
		icon, iconStyle := StateIcon(it.Status)
		row := Row{
			Cell{Text: icon, Style: iconStyle},
			Cell{Text: it.Name},
			Cell{Text: it.Type},
			Cell{Text: it.Region},
			Cell{Text: it.Info, Style: &InfoStyle},
		}
		rows = append(rows, row)
	}
	return rows
}
