package slices

func PartitionStrings(slice []interface{}) ([]string, []interface{}) {
	var (
		strings []string
		rest    []interface{}
	)

	for _, intf := range slice {
		switch v := intf.(type) {
		case string:
			strings = append(strings, v)
		default:
			rest = append(rest, v)
		}
	}

	return strings, rest
}
