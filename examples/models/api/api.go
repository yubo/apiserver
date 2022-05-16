package api

type Secret struct {
	Name string `json:"name" sql:",where"`
	Data string `json:"data"`
}
