package feishu

// CardTemplate defines a Feishu interactive card template
type CardTemplate struct {
	Config   *CardConfig   `json:"config,omitempty"`
	Header   *CardHeader   `json:"header,omitempty"`
	Elements []CardElement `json:"elements,omitempty"`
	Cards    []CardBlock   `json:"cards,omitempty"`
}

// CardConfig defines card configuration
type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode,omitempty"`
	EnableForward  bool `json:"enable_forward,omitempty"`
}

// CardHeader defines card header
type CardHeader struct {
	Template string `json:"template,omitempty"`
	Title    *Text  `json:"title,omitempty"`
}

// CardElement represents a card element (markdown, note, alert, etc.)
type CardElement struct {
	Type     string        `json:"tag"`
	Text     *Text         `json:"text,omitempty"`
	Content  string        `json:"content,omitempty"`
	Elements []CardElement `json:"elements,omitempty"`
	Actions  []CardAction  `json:"actions,omitempty"`
}

// CardBlock represents a card block
type CardBlock struct {
	Type     string        `json:"tag"`
	Text     *Text         `json:"text,omitempty"`
	Content  string        `json:"content,omitempty"`
	Elements []CardElement `json:"elements,omitempty"`
	Actions  []CardAction  `json:"actions,omitempty"`
}

// Text defines text content with escaping
type Text struct {
	Content string `json:"content,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

// CardAction defines an interactive action (button, etc.)
type CardAction struct {
	Type  string      `json:"tag"`
	Text  *Text       `json:"text,omitempty"`
	Value interface{} `json:"value,omitempty"`
	URL   string      `json:"url,omitempty"`
}

// Card colors (templates)
const (
	CardTemplateBlue   = "blue"
	CardTemplateWathet = "wathet"
	CardTemplateGreen  = "green"
	CardTemplateYellow = "yellow"
	CardTemplateOrange = "orange"
	CardTemplateRed    = "red"
	Purple             = "purple"
	Grey               = "grey"
)

// Text types
const (
	TextTypePlainText = "plain_text"
	TextTypeLarkMD    = "lark_md"
)

// Element types
const (
	ElementDiv      = "div"
	ElementNote     = "note"
	ElementAlert    = "alert"
	ElementAction   = "action"
	ElementButton   = "button"
	ElementMarkdown = "markdown"
)

// Button types
const (
	ButtonTypeDefault = "default"
	ButtonTypePrimary = "primary"
	ButtonTypeDanger  = "danger"
)
