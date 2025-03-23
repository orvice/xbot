package conf

type OpenAI struct {
	Endpoint string   `yaml:"endpoint"`
	Key      string   `yaml:"key"`
	Model    string   `yaml:"model"`
	Keys     []string `yaml:"keys"`
}

type Config struct {
	TelegramBotToken string `yaml:"telegramBotToken"`

	SummaryModels []string `yaml:"summaryModels"`
	Host          string   `yaml:"host"`
	DBName        string   `yaml:"dbName"`
	OpenAI        OpenAI   `yaml:"openAI"`
	PictureVendor OpenAI   `yaml:"pictureVendor"`
	Bots          []Bot    `yaml:"bots"`
}

type Bot struct {
	Name   string
	Token  string
	Enable bool
}

var (
	Conf = new(Config)
)

func (c *Config) Print() {}
