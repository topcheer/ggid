// Package scim implements RFC 7644 PATCH path engine for SCIM resources.
// Supports add, replace, and remove operations on nested and multi-valued attributes.
package scim

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PatchOperation represents a single SCIM PATCH operation.
type PatchOperation struct {
	Op    string          `json:"op"`
	Path  string          `json:"path,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// PatchRequest is the top-level SCIM PATCH request body.
type PatchRequest struct {
	Schemas    []string         `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

// ApplyPatch applies a list of PATCH operations to a SCIM resource attribute map.
// The attrs map is the mutable representation of the resource's attributes.
// Returns the updated attrs map.
func ApplyPatch(attrs map[string]any, ops []PatchOperation) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range attrs {
		result[k] = v
	}

	for i, op := range ops {
		opLower := strings.ToLower(strings.TrimSpace(op.Op))
		switch opLower {
		case "add":
			if err := applyAdd(result, op.Path, op.Value); err != nil {
				return nil, fmt.Errorf("op[%d] add: %w", i, err)
			}
		case "replace":
			if err := applyReplace(result, op.Path, op.Value); err != nil {
				return nil, fmt.Errorf("op[%d] replace: %w", i, err)
			}
		case "remove":
			if err := applyRemove(result, op.Path); err != nil {
				return nil, fmt.Errorf("op[%d] remove: %w", i, err)
			}
		default:
			return nil, fmt.Errorf("op[%d] unsupported operation %q", i, op.Op)
		}
	}

	return result, nil
}

// applyAdd handles the SCIM "add" operation.
func applyAdd(attrs map[string]any, path string, value json.RawMessage) error {
	var val any
	if len(value) > 0 {
		if err := json.Unmarshal(value, &val); err != nil {
			return fmt.Errorf("invalid value: %w", err)
		}
	}

	// No path: merge value object into attrs
	if path == "" {
		valMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("add without path requires object value")
		}
		for k, v := range valMap {
			setAttrCaseInsensitive(attrs, k, v)
		}
		return nil
	}

	attrName, subPath, filter := parsePatchPath(path)

	// No filter on path
	if filter == "" {
		if subPath != "" {
			return setNestedAttr(attrs, attrName, subPath, val)
		}

		// Check if existing value is an array (multi-valued attribute)
		existing := getAttrCaseInsensitive(attrs, attrName)
		if existingArr, ok := existing.([]any); ok {
			if newValArr, ok := val.([]any); ok {
				existingArr = append(existingArr, newValArr...)
			} else {
				existingArr = append(existingArr, val)
			}
			setAttrCaseInsensitive(attrs, attrName, existingArr)
			return nil
		}

		setAttrCaseInsensitive(attrs, attrName, val)
		return nil
	}

	// Has filter on multi-valued attribute
	return addOrReplaceInArray(attrs, attrName, filter, subPath, val, false)
}

// applyReplace handles the SCIM "replace" operation.
func applyReplace(attrs map[string]any, path string, value json.RawMessage) error {
	var val any
	if len(value) > 0 {
		if err := json.Unmarshal(value, &val); err != nil {
			return fmt.Errorf("invalid value: %w", err)
		}
	}

	if path == "" {
		valMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("replace without path requires object value")
		}
		for k, v := range valMap {
			setAttrCaseInsensitive(attrs, k, v)
		}
		return nil
	}

	attrName, subPath, filter := parsePatchPath(path)

	if filter == "" {
		if subPath != "" {
			return setNestedAttr(attrs, attrName, subPath, val)
		}
		// If both existing and new values are maps, merge (SCIM RFC 7644 §3.5.2.3
		// replace on complex attribute updates specified keys, preserves others).
		if existing := getAttrCaseInsensitive(attrs, attrName); existing != nil {
			if existMap, ok := existing.(map[string]any); ok {
				if newMap, ok := val.(map[string]any); ok {
					for k, v := range newMap {
						setAttrCaseInsensitive(existMap, k, v)
					}
					return nil // merged in-place
				}
			}
		}
		setAttrCaseInsensitive(attrs, attrName, val)
		return nil
	}

	return addOrReplaceInArray(attrs, attrName, filter, subPath, val, true)
}

// applyRemove handles the SCIM "remove" operation.
func applyRemove(attrs map[string]any, path string) error {
	if path == "" {
		return fmt.Errorf("remove requires a path")
	}

	attrName, subPath, filter := parsePatchPath(path)

	if filter == "" {
		if subPath != "" {
			parent := getAttrCaseInsensitive(attrs, attrName)
			if m, ok := parent.(map[string]any); ok {
				deleteCaseInsensitive(m, subPath)
			}
			return nil
		}
		deleteCaseInsensitive(attrs, attrName)
		return nil
	}

	// Remove elements matching filter from array
	existing := getAttrCaseInsensitive(attrs, attrName)
	items, ok := existing.([]any)
	if !ok {
		return nil
	}

	innerFilter, err := ParseFilter(filter)
	if err != nil {
		return fmt.Errorf("invalid filter in path: %w", err)
	}

	var kept []any
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if innerFilter != nil && innerFilter.Evaluate(m) {
				if subPath != "" {
					deleteCaseInsensitive(m, subPath)
				}
				continue // skip this item (remove it)
			}
		}
		kept = append(kept, item)
	}
	setAttrCaseInsensitive(attrs, attrName, kept)
	return nil
}

// parsePatchPath parses a SCIM PATCH path into components.
// Example: emails[type eq "work"].value
// Returns: attrName="emails", subPath="value", filter="type eq \"work\""
//
// Supports URN paths in two notations:
//   - Dot notation:   urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.department
//   - Colon notation: urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department (RFC 7644)
//
// For URN paths, the "." in version numbers (e.g., "2.0") is NOT treated as
// a separator. Only a "." or ":" appearing after the schema type suffix
// (the segment after the last ":") is treated as a sub-path separator.
func parsePatchPath(path string) (attrName, subPath, filter string) {
	path = strings.TrimSpace(path)

	bracketIdx := strings.Index(path, "[")
	if bracketIdx < 0 {
		// No filter - simple, dotted, or URN path
		// For URN paths (e.g., "urn:...:2.0:User"), the colons and the "."
		// in version numbers are part of the schema identifier, not path
		// separators. Only split on a "." or ":" that appears AFTER the last ":"
		// in the URN prefix.
		if strings.HasPrefix(path, "urn:") {
			// Split into segments to find the schema/type boundary.
			// SCIM URN format: urn:...:version:TypeSuffix:subAttr:subSubAttr
			// e.g., urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department
			segments := strings.Split(path, ":")

			// Find version segment (contains ".") followed by type suffix (User, Group).
			for i := 0; i < len(segments)-1; i++ {
				if strings.Contains(segments[i], ".") && isAlphaStr(segments[i+1]) {
					// segments[i] = version (2.0), segments[i+1] = type suffix (User)
					if i+2 < len(segments) {
						// Sub-attributes exist after type suffix (colon notation)
						schemaUrn := strings.Join(segments[:i+2], ":")
						subPath := strings.Join(segments[i+2:], ":")
						return schemaUrn, subPath, ""
					}
					// No sub-attributes — entire path is the schema URN
					return path, "", ""
				}
			}

			// Fallback: dot notation (urn:...:User.department)
			lastColon := strings.LastIndex(path, ":")
			afterColon := path[lastColon+1:]
			if dotIdx := strings.Index(afterColon, "."); dotIdx >= 0 {
				splitAt := lastColon + 1 + dotIdx
				return path[:splitAt], path[splitAt+1:], ""
			}
			return path, "", ""
		}
		if dotIdx := strings.Index(path, "."); dotIdx >= 0 {
			return path[:dotIdx], path[dotIdx+1:], ""
		}
		return path, "", ""
	}

	// Has filter
	attrName = path[:bracketIdx]
	closeIdx := strings.Index(path[bracketIdx:], "]")
	if closeIdx < 0 {
		return attrName, "", ""
	}
	filter = strings.TrimSpace(path[bracketIdx+1 : bracketIdx+closeIdx])

	// Check for sub-path after ]
	rest := path[bracketIdx+closeIdx+1:]
	if strings.HasPrefix(rest, ".") {
		subPath = rest[1:]
	} else if rest != "" {
		subPath = rest
	}

	return attrName, subPath, filter
}

// setNestedAttr sets a nested attribute value: parent.child = value
// Supports multi-level dotted paths: parent.child.grandchild = value
func setNestedAttr(attrs map[string]any, parent, child string, value any) error {
	existing := getAttrCaseInsensitive(attrs, parent)
	var m map[string]any
	if existing == nil {
		m = make(map[string]any)
	} else if em, ok := existing.(map[string]any); ok {
		m = em
	} else {
		m = make(map[string]any)
	}

	// Navigate dotted sub-path (e.g., "manager.displayName")
	parts := strings.Split(child, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			// Leaf — set the value
			setAttrCaseInsensitive(current, part, value)
		} else {
			// Intermediate — navigate or create nested map
			next := getAttrCaseInsensitive(current, part)
			var nm map[string]any
			if nmm, ok := next.(map[string]any); ok {
				nm = nmm
			} else {
				nm = make(map[string]any)
			}
			setAttrCaseInsensitive(current, part, nm)
			current = nm
		}
	}

	setAttrCaseInsensitive(attrs, parent, m)
	return nil
}

// addOrReplaceInArray operates on multi-valued attributes with a filter.
func addOrReplaceInArray(attrs map[string]any, attrName, filter, subPath string, value any, isReplace bool) error {
	existing := getAttrCaseInsensitive(attrs, attrName)
	items, _ := existing.([]any)
	if items == nil {
		items = []any{}
	}

	innerFilter, err := ParseFilter(filter)
	if err != nil {
		return fmt.Errorf("invalid filter %q: %w", filter, err)
	}

	matched := false
	for i, item := range items {
		if m, ok := item.(map[string]any); ok {
			if innerFilter == nil || innerFilter.Evaluate(m) {
				matched = true
				if subPath != "" {
					setAttrCaseInsensitive(m, subPath, value)
				} else {
					// Replace entire matching element
					if valMap, ok := value.(map[string]any); ok {
						for k, v := range valMap {
							setAttrCaseInsensitive(m, k, v)
						}
					}
				}
				items[i] = m
			}
		}
	}

	// For add: if no items matched, append the new value
	if !matched && !isReplace {
		if valMap, ok := value.(map[string]any); ok {
			items = append(items, valMap)
		} else {
			items = append(items, value)
		}
	}

	setAttrCaseInsensitive(attrs, attrName, items)
	return nil
}

// --- Helpers ---

func getAttrCaseInsensitive(attrs map[string]any, key string) any {
	for k, v := range attrs {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

func setAttrCaseInsensitive(attrs map[string]any, key string, value any) {
	for k := range attrs {
		if strings.EqualFold(k, key) {
			attrs[k] = value
			return
		}
	}
	attrs[key] = value
}

func deleteCaseInsensitive(attrs map[string]any, key string) {
	for k := range attrs {
		if strings.EqualFold(k, key) {
			delete(attrs, k)
			return
		}
	}
}

// isAlphaStr returns true if s contains only ASCII alphabetic characters.
// Used to distinguish URN type suffixes (User, Group) from version numbers (2.0).
func isAlphaStr(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

// PatchedAttrsToSCIMUser builds a SCIMUser from patched attrs map.
func PatchedAttrsToSCIMUser(attrs map[string]any) SCIMUser {
	u := SCIMUser{
		Schemas: []string{scimUserSchema},
	}

	if v := getAttrCaseInsensitive(attrs, "userName"); v != nil {
		u.UserName = toStr(v)
	}
	if v := getAttrCaseInsensitive(attrs, "displayName"); v != nil {
		u.DisplayName = toStr(v)
	}
	if v := getAttrCaseInsensitive(attrs, "active"); v != nil {
		u.Active = toStr(v) == "true"
	}
	if v := getAttrCaseInsensitive(attrs, "id"); v != nil {
		u.ID = toStr(v)
	}
	if v := getAttrCaseInsensitive(attrs, "externalId"); v != nil {
		u.ExternalID = toStr(v)
	}

	// Handle name sub-object
	if nameMap := getAttrCaseInsensitive(attrs, "name"); nameMap != nil {
		if m, ok := nameMap.(map[string]any); ok {
			u.Name.GivenName = toStr(getAttrCaseInsensitive(m, "givenName"))
			u.Name.FamilyName = toStr(getAttrCaseInsensitive(m, "familyName"))
		}
	}

	// Handle emails
	if emailsVal := getAttrCaseInsensitive(attrs, "emails"); emailsVal != nil {
		if arr, ok := emailsVal.([]any); ok {
			for _, e := range arr {
				if em, ok := e.(map[string]any); ok {
					se := SCIMEmail{
						Value: toStr(getAttrCaseInsensitive(em, "value")),
						Type:  toStr(getAttrCaseInsensitive(em, "type")),
					}
					if p := getAttrCaseInsensitive(em, "primary"); p != nil {
						se.Primary = toStr(p) == "true"
					}
					u.Emails = append(u.Emails, se)
				}
			}
		}
	}

	// Handle phone numbers
	if phonesVal := getAttrCaseInsensitive(attrs, "phoneNumbers"); phonesVal != nil {
		if arr, ok := phonesVal.([]any); ok {
			for _, p := range arr {
				if pm, ok := p.(map[string]any); ok {
					sp := SCIMPhone{
						Value: toStr(getAttrCaseInsensitive(pm, "value")),
						Type:  toStr(getAttrCaseInsensitive(pm, "type")),
					}
					u.PhoneNumbers = append(u.PhoneNumbers, sp)
				}
			}
		}
	}

	return u
}
