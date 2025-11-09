package locales

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed ru.json
var rawRU []byte

type Messages struct {
	MainMenuText                   string   `json:"main_menu_text"`
	MainMenuButtons                []string `json:"main_menu_buttons"`
	ProfileTitle                   string   `json:"profile_title"`
	ProfileSkillsTitle             string   `json:"profile_skills_title"`
	ProfileLevelBalanceTemplate    string   `json:"profile_level_balance_template"`
	ProfileHistoryButton           string   `json:"profile_history_button"`
	ProfileEditButton              string   `json:"profile_edit_button"`
	ProfileSecurityButton          string   `json:"profile_security_button"`
	ProfileBackButton              string   `json:"profile_back_button"`
	ProfileCoinsButton             string   `json:"profile_coins_button"`
	ProfileSecurityTitle           string   `json:"profile_security_title"`
	ProfileSecurityText            string   `json:"profile_security_text"`
	ProfileSecuritySOSButton       string   `json:"profile_security_sos_button"`
	ProfileSecuritySOSLink         string   `json:"profile_security_sos_link"`
	ProfileHistoryText             string   `json:"profile_history_text"`
	ProfileEditText                string   `json:"profile_edit_text"`
	RegistrationStartText          string   `json:"registration_start_text"`
	RegistrationAgeRetryText       string   `json:"registration_age_retry_text"`
	RegistrationAgeUnder18Button   string   `json:"registration_age_under_18_button"`
	RegistrationAge18_24Button     string   `json:"registration_age_18_24_button"`
	RegistrationAge25_34Button     string   `json:"registration_age_25_34_button"`
	RegistrationAge35_44Button     string   `json:"registration_age_35_44_button"`
	RegistrationAge45_54Button     string   `json:"registration_age_45_54_button"`
	RegistrationAge55_64Button     string   `json:"registration_age_55_64_button"`
	RegistrationAge65PlusButton    string   `json:"registration_age_65_plus_button"`
	RegistrationSexPrompt          string   `json:"registration_sex_prompt"`
	RegistrationSexMaleText        string   `json:"registration_sex_male_text"`
	RegistrationSexFemaleText      string   `json:"registration_sex_female_text"`
	RegistrationLocationPrompt     string   `json:"registration_location_prompt"`
	RegistrationLocationGeoButton  string   `json:"registration_location_geo_button"`
	RegistrationLocationSkipButton string   `json:"registration_location_skip_button"`
	RegistrationLocationRetryText  string   `json:"registration_location_retry_text"`
	RegistrationAboutPrompt        string   `json:"registration_about_prompt"`
	RegistrationAboutConfirmButton string   `json:"registration_about_confirm_button"`
	RegistrationAboutOptions       []string `json:"registration_about_options"`
	RegistrationErrorText          string   `json:"registration_error_text"`
	RegistrationCompleteText       string   `json:"registration_complete_text"`
	NewUserWelcomeText             string   `json:"new_user_welcome_text"`
	NewUserJoinButton              string   `json:"new_user_join_button"`
	CoinsIntroText                 string   `json:"coins_intro_text"`
	CoinsButtons                   []string `json:"coins_buttons"`
	CoinsHowToGetText              string   `json:"coins_how_to_get_text"`
	CoinsHowToSpendText            string   `json:"coins_how_to_spend_text"`
	CoinsLevelsText                string   `json:"coins_levels_text"`
	CoinsBackButton                string   `json:"coins_back_button"`
	AboutDobrikaText               string   `json:"about_dobrika_text"`
	AboutDobrikaButtons            []string `json:"about_dobrika_buttons"`
	AboutDobrikaHowText            string   `json:"about_dobrika_how_text"`
	AboutDobrikaRulesText          string   `json:"about_dobrika_rules_text"`
	AboutDobrikaInitiatorText      string   `json:"about_dobrika_initiator_text"`
	AboutDobrikaSupportText        string   `json:"about_dobrika_support_text"`
}

var (
	once    sync.Once
	cached  Messages
	loadErr error
)

func Load() (Messages, error) {
	once.Do(func() {
		defaults := defaultMessages()

		var overrides Messages
		if err := json.Unmarshal(rawRU, &overrides); err != nil {
			loadErr = fmt.Errorf("failed to unmarshal ru.json: %w", err)
			cached = defaults
			return
		}

		cached = mergeMessages(defaults, overrides)
	})
	return cached, loadErr
}

func mergeMessages(base, overrides Messages) Messages {
	if overrides.MainMenuText != "" {
		base.MainMenuText = overrides.MainMenuText
	}
	if len(overrides.MainMenuButtons) > 0 {
		base.MainMenuButtons = overrides.MainMenuButtons
	}
	if overrides.ProfileTitle != "" {
		base.ProfileTitle = overrides.ProfileTitle
	}
	if overrides.ProfileSkillsTitle != "" {
		base.ProfileSkillsTitle = overrides.ProfileSkillsTitle
	}
	if overrides.ProfileLevelBalanceTemplate != "" {
		base.ProfileLevelBalanceTemplate = overrides.ProfileLevelBalanceTemplate
	}
	if overrides.ProfileHistoryButton != "" {
		base.ProfileHistoryButton = overrides.ProfileHistoryButton
	}
	if overrides.ProfileEditButton != "" {
		base.ProfileEditButton = overrides.ProfileEditButton
	}
	if overrides.ProfileSecurityButton != "" {
		base.ProfileSecurityButton = overrides.ProfileSecurityButton
	}
	if overrides.ProfileBackButton != "" {
		base.ProfileBackButton = overrides.ProfileBackButton
	}
	if overrides.ProfileCoinsButton != "" {
		base.ProfileCoinsButton = overrides.ProfileCoinsButton
	}
	if overrides.ProfileHistoryText != "" {
		base.ProfileHistoryText = overrides.ProfileHistoryText
	}
	if overrides.ProfileEditText != "" {
		base.ProfileEditText = overrides.ProfileEditText
	}
	if overrides.ProfileSecurityTitle != "" {
		base.ProfileSecurityTitle = overrides.ProfileSecurityTitle
	}
	if overrides.ProfileSecurityText != "" {
		base.ProfileSecurityText = overrides.ProfileSecurityText
	}
	if overrides.ProfileSecuritySOSButton != "" {
		base.ProfileSecuritySOSButton = overrides.ProfileSecuritySOSButton
	}
	if overrides.ProfileSecuritySOSLink != "" {
		base.ProfileSecuritySOSLink = overrides.ProfileSecuritySOSLink
	}
	if overrides.RegistrationStartText != "" {
		base.RegistrationStartText = overrides.RegistrationStartText
	}
	if overrides.RegistrationAgeRetryText != "" {
		base.RegistrationAgeRetryText = overrides.RegistrationAgeRetryText
	}
	if overrides.RegistrationAgeUnder18Button != "" {
		base.RegistrationAgeUnder18Button = overrides.RegistrationAgeUnder18Button
	}
	if overrides.RegistrationAge18_24Button != "" {
		base.RegistrationAge18_24Button = overrides.RegistrationAge18_24Button
	}
	if overrides.RegistrationAge25_34Button != "" {
		base.RegistrationAge25_34Button = overrides.RegistrationAge25_34Button
	}
	if overrides.RegistrationAge35_44Button != "" {
		base.RegistrationAge35_44Button = overrides.RegistrationAge35_44Button
	}
	if overrides.RegistrationAge45_54Button != "" {
		base.RegistrationAge45_54Button = overrides.RegistrationAge45_54Button
	}
	if overrides.RegistrationAge55_64Button != "" {
		base.RegistrationAge55_64Button = overrides.RegistrationAge55_64Button
	}
	if overrides.RegistrationAge65PlusButton != "" {
		base.RegistrationAge65PlusButton = overrides.RegistrationAge65PlusButton
	}
	if overrides.RegistrationSexPrompt != "" {
		base.RegistrationSexPrompt = overrides.RegistrationSexPrompt
	}
	if overrides.RegistrationSexMaleText != "" {
		base.RegistrationSexMaleText = overrides.RegistrationSexMaleText
	}
	if overrides.RegistrationSexFemaleText != "" {
		base.RegistrationSexFemaleText = overrides.RegistrationSexFemaleText
	}
	if overrides.RegistrationLocationPrompt != "" {
		base.RegistrationLocationPrompt = overrides.RegistrationLocationPrompt
	}
	if overrides.RegistrationLocationGeoButton != "" {
		base.RegistrationLocationGeoButton = overrides.RegistrationLocationGeoButton
	}
	if overrides.RegistrationLocationSkipButton != "" {
		base.RegistrationLocationSkipButton = overrides.RegistrationLocationSkipButton
	}
	if overrides.RegistrationLocationRetryText != "" {
		base.RegistrationLocationRetryText = overrides.RegistrationLocationRetryText
	}
	if overrides.RegistrationAboutPrompt != "" {
		base.RegistrationAboutPrompt = overrides.RegistrationAboutPrompt
	}
	if overrides.RegistrationAboutConfirmButton != "" {
		base.RegistrationAboutConfirmButton = overrides.RegistrationAboutConfirmButton
	}
	if len(overrides.RegistrationAboutOptions) > 0 {
		base.RegistrationAboutOptions = overrides.RegistrationAboutOptions
	}
	if overrides.RegistrationErrorText != "" {
		base.RegistrationErrorText = overrides.RegistrationErrorText
	}
	if overrides.RegistrationCompleteText != "" {
		base.RegistrationCompleteText = overrides.RegistrationCompleteText
	}
	if overrides.NewUserWelcomeText != "" {
		base.NewUserWelcomeText = overrides.NewUserWelcomeText
	}
	if overrides.NewUserJoinButton != "" {
		base.NewUserJoinButton = overrides.NewUserJoinButton
	}
	if overrides.CoinsIntroText != "" {
		base.CoinsIntroText = overrides.CoinsIntroText
	}
	if len(overrides.CoinsButtons) > 0 {
		base.CoinsButtons = overrides.CoinsButtons
	}
	if overrides.CoinsHowToGetText != "" {
		base.CoinsHowToGetText = overrides.CoinsHowToGetText
	}
	if overrides.CoinsHowToSpendText != "" {
		base.CoinsHowToSpendText = overrides.CoinsHowToSpendText
	}
	if overrides.CoinsLevelsText != "" {
		base.CoinsLevelsText = overrides.CoinsLevelsText
	}
	if overrides.CoinsBackButton != "" {
		base.CoinsBackButton = overrides.CoinsBackButton
	}
	if overrides.AboutDobrikaText != "" {
		base.AboutDobrikaText = overrides.AboutDobrikaText
	}
	if len(overrides.AboutDobrikaButtons) > 0 {
		base.AboutDobrikaButtons = overrides.AboutDobrikaButtons
	}
	if overrides.AboutDobrikaHowText != "" {
		base.AboutDobrikaHowText = overrides.AboutDobrikaHowText
	}
	if overrides.AboutDobrikaRulesText != "" {
		base.AboutDobrikaRulesText = overrides.AboutDobrikaRulesText
	}
	if overrides.AboutDobrikaInitiatorText != "" {
		base.AboutDobrikaInitiatorText = overrides.AboutDobrikaInitiatorText
	}
	if overrides.AboutDobrikaSupportText != "" {
		base.AboutDobrikaSupportText = overrides.AboutDobrikaSupportText
	}

	return base
}

func defaultMessages() Messages {
	return Messages{
		MainMenuText: "Главное меню. Что хотите сделать?",
		MainMenuButtons: []string{
			"Хочу помочь",
			"Мне нужна помощь",
			"Мой профиль",
			"О Добрике",
		},
		ProfileTitle:                "👤 *Мой профиль*",
		ProfileSkillsTitle:          "Навыки и интересы:",
		ProfileLevelBalanceTemplate: "🎖 Уровень: *%s*\n💰 Репутация: *%d* добриков",
		ProfileHistoryButton:        "📜 История дел",
		ProfileEditButton:           "✏️ Редактировать",
		ProfileSecurityButton:       "🛡 Безопасность",
		ProfileBackButton:           "⬅️ Назад в меню",
		ProfileCoinsButton:          "💰 Добрики",
		ProfileHistoryText:          "История добрых дел появится совсем скоро 💚",
		ProfileEditText:             "Редактирование профиля появится в ближайшем обновлении.",
		ProfileSecurityTitle:        "🛡 Безопасность встреч офлайн",
		ProfileSecurityText:         "• Назначайте встречи только в людных местах\n• Делитесь планами с близкими\n• Пользуйтесь кнопкой SOS в экстренных ситуациях\n\nВсе правила и контакты: %s",
		ProfileSecuritySOSButton:    "🚨 Открыть памятку",
		ProfileSecuritySOSLink:      "https://dobrika.example/safety",
		AboutDobrikaText:            "Добрика — бот добрых дел. Здесь можно помогать другим и получать добрики за сделанное добро.",
		AboutDobrikaButtons: []string{
			"💚 Как это работает",
			"🧭 Правила и безопасность",
			"🏢 Стать инициатором",
			"📞 Связаться с поддержкой",
			"⬅️ Назад в меню",
		},
		AboutDobrikaHowText:            "1. Выбери доброе дело рядом или онлайн.\n2. Выполни его и отправь подтверждение.\n3. Получи добрики и расти в Добрике!",
		AboutDobrikaRulesText:          "Следуй простым правилам безопасности: назначай встречи в людных местах, предупреждай родных и бери с собой телефон.",
		AboutDobrikaInitiatorText:      "Хочешь разместить доброе дело или инициативу? Напиши нам — поможем организовать и привлечь волонтёров.",
		AboutDobrikaSupportText:        "Служба поддержки Добрики отвечает в рабочее время. Пиши на support@dobrika.example или в чат @dobrika_support.",
		RegistrationStartText:          "🎂 Укажите ваш возраст:",
		RegistrationAgeRetryText:       "Выберите вариант на клавиатуре или укажите число.",
		RegistrationAgeUnder18Button:   "< 18 лет",
		RegistrationAge18_24Button:     "18–24 года",
		RegistrationAge25_34Button:     "25–34 года",
		RegistrationAge35_44Button:     "35–44 года",
		RegistrationAge45_54Button:     "45–54 года",
		RegistrationAge55_64Button:     "55–64 года",
		RegistrationAge65PlusButton:    "65+ лет",
		RegistrationSexPrompt:          "Выберите пол:",
		RegistrationSexMaleText:        "Мужчина",
		RegistrationSexFemaleText:      "Женщина",
		RegistrationLocationPrompt:     "Где вы сейчас находитесь? Можно отправить геопозицию или ответить текстом.",
		RegistrationLocationGeoButton:  "Отправить геолокацию",
		RegistrationLocationSkipButton: "Не хочу делиться",
		RegistrationLocationRetryText:  "Не смог получить локацию. Попробуйте ещё раз или отправьте название города текстом.",
		RegistrationAboutPrompt:        "Расскажите, как вы готовы помогать",
		RegistrationAboutConfirmButton: "Подтвердить выбор",
		RegistrationAboutOptions: []string{
			"🛒 Магазин",
			"💬 Разговор",
			"👩‍💻 Помогу удаленно",
			"📦 Доставить",
			"📚 Учеба",
			"🧹 Помогу по дому",
			"🚗 Подвезти",
			"💰 Деньгами",
			"🐾 Питомцы",
			"🤷‍♂️ Не знаю",
		},
		RegistrationErrorText:    "Не удалось сохранить данные. Попробуйте позже.",
		RegistrationCompleteText: "Спасибо! Мы сохранили ваши данные.",
		NewUserWelcomeText:       "Добро пожаловать! Нажмите, чтобы присоединиться.",
		NewUserJoinButton:        "Присоединиться",
		CoinsIntroText:           "Добрики — благодарность за твою помощь. Чем больше добрых дел, тем больше добриков и выше уровень.",
		CoinsButtons: []string{
			"Как получить",
			"На что потратить",
			"Уровни",
			"⬅️ Назад в профиль",
		},
		CoinsHowToGetText:   "Получай добрики, выполняя задания, помогая людям и подтверждая добрые дела.",
		CoinsHowToSpendText: "Добрики можно обменять на сувениры, участвовать в челленджах и дарить друзьям.",
		CoinsLevelsText:     "Каждый уровень открывает новые задания и показывает твою активность в сообществе.",
		CoinsBackButton:     "⬅️ Назад в профиль",
	}
}
