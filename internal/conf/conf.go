package conf

type OpenAI struct {
	Endpoint string   `yaml:"endpoint"`
	Key      string   `yaml:"key"`
	Model    string   `yaml:"model"`
	Keys     []string `yaml:"keys"`
}

type Config struct {
	TelegramBotToken string `yaml:"telegramBotToken"`

	ChatEndpoint string `yaml:"chatEndpoint"`

	SummaryModels []string `yaml:"summaryModels"`
	Host          string   `yaml:"host"`
	DBName        string   `yaml:"dbName"`
	OpenAI        OpenAI   `yaml:"openAI"`
	PictureVendor OpenAI   `yaml:"pictureVendor"`
	Bots          []Bot    `yaml:"bots"`
	S3            S3Config `yaml:"s3Config"`

	MessageStorage string `yaml:"messageStorage"`
}

type Bot struct {
	Name   string
	Token  string
	Enable bool
}

type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
}

var (
	Conf = new(Config)
)

func (c *Config) Print() {}
