package types

type UserSearchResult struct {
	UserID      string            `json:"userId"`
	DisplayName string            `json:"displayName"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type UserSearchResultDangerous struct {
	UserID      string                 `json:"userId"`
	DisplayName string                 `json:"displayName"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Email       string                 `json:"email,omitempty"`
}
