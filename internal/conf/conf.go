package conf

type OpenAI struct {
	Endpoint string `yaml:"endpoint"`
	Key      string `yaml:"key"`
	Model    string `yaml:"model"`
}

type Config struct {
	TelegramBotToken string `yaml:"telegramBotToken"`
	OpenAI           OpenAI `yaml:"openAI"`
}

var (
	Conf = new(Config)
)

func (c *Config) Print() {}
