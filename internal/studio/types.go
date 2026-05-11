package studio

type Endpoint struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type APIKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	Endpoint  string `json:"endpoint"`
	AK        string `json:"ak,omitempty"`
	SK        string `json:"sk,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type KeyStoreData struct {
	Active string   `json:"active"`
	Keys   []APIKey `json:"keys"`
}

type MaskedKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Preview   string `json:"preview"`
	Endpoint  string `json:"endpoint"`
	HasAkSk   bool   `json:"hasAkSk"`
	AKPreview string `json:"akPreview"`
	CreatedAt string `json:"createdAt"`
}

type KeyListResponse struct {
	Active    string              `json:"active"`
	Endpoints map[string]Endpoint `json:"endpoints"`
	Keys      []MaskedKey         `json:"keys"`
}

type AddKeyRequest struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Endpoint string `json:"endpoint"`
	AK       string `json:"ak"`
	SK       string `json:"sk"`
}

type UpdateKeyRequest struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

type Selection struct {
	UserPrompt   string            `json:"userPrompt"`
	Model        string            `json:"model"`
	Duration     float64           `json:"duration"`
	SoundOn      *bool             `json:"soundOn"`
	AspectRatio  *SelectionField   `json:"aspectRatio"`
	Resolution   *SelectionField   `json:"resolution"`
	CameraMotion *SelectionPrompt  `json:"cameraMotion"`
	Camera       *SelectionPrompt  `json:"camera"`
	Lens         *SelectionPrompt  `json:"lens"`
	ColorGrading *SelectionPrompt  `json:"colorGrading"`
	Genre        *SelectionPrompt  `json:"genre"`
	FirstFrame   *DataRef          `json:"firstFrame"`
	LastFrame    *DataRef          `json:"lastFrame"`
	RefImages    []DataRef         `json:"refImages"`
	RefVideos    []DataRef         `json:"refVideos"`
	RefAudios    []DataRef         `json:"refAudios"`
}

type SelectionField struct {
	Value string `json:"value"`
}

type SelectionPrompt struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

type DataRef struct {
	DataUrl string `json:"dataUrl"`
}

type SeedreamRequest struct {
	Prompt          string   `json:"prompt"`
	Model           string   `json:"model"`
	Size            string   `json:"size"`
	Seed            *int     `json:"seed"`
	ReferenceImages []string `json:"referenceImages"`
}

type EmptySuccess struct {
	Ok      bool   `json:"ok"`
	Deleted string `json:"deleted,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
