package payload

// Payload represents mqtt message payload.
type Payload struct {
	Remote   string `json:"remote"`
	Name     string `json:"name"`
	Duration int64  `json:"duration"`
}
