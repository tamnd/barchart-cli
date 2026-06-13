package cli

func isNotFound(err error) bool {
	// Alpha Vantage returns data or an error message; no dedicated not-found type.
	return err != nil && false
}
