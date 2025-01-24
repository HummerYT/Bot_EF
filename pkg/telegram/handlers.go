package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strconv"
	"sync"
)

type Bot struct {
	bot            *tgbotapi.BotAPI
	startMenu      tgbotapi.InlineKeyboardMarkup
	physicsMenu    tgbotapi.InlineKeyboardMarkup
	difficultyMenu tgbotapi.InlineKeyboardMarkup
	nextTaskMenu   tgbotapi.InlineKeyboardMarkup
	lastCallback   string
	userStates     map[int64]UserState
	mu             sync.Mutex
}

type UserState struct {
	Section    string
	Difficulty string
	Answer     int64
	Next       string
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	startMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Продолжить", "choose_physics"),
		),
	)

	physicsMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Динамика", "dynamics"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Механика", "mechanics"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Электростатика", "electrostatics"),
		),
	)

	difficultyMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начинающий", "beginner"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Продвинутый", "advanced"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Знаток", "expert"),
		),
	)

	nextTaskMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, пришли мне другую задачу.", "next_task"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Нет. Выбрать другой раздел физики.", "choose_physics"),
		),
	)

	return &Bot{
		bot:            bot,
		startMenu:      startMenu,
		physicsMenu:    physicsMenu,
		difficultyMenu: difficultyMenu,
		nextTaskMenu:   nextTaskMenu,
		userStates:     make(map[int64]UserState),
	}
}

func (b *Bot) Start() error {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)
	updates, err := b.initUpdatesChannel()
	if err != nil {
		return err
	}
	b.handleUpdates(updates)
	return nil
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				b.handleCommand(update.Message)
				continue
			}
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			if update.CallbackQuery.ID != b.lastCallback {
				b.lastCallback = update.CallbackQuery.ID
				b.callbacks(update)
			}
		}
	}
}

func (b *Bot) initUpdatesChannel() (tgbotapi.UpdatesChannel, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.bot.GetUpdatesChan(u)
}

const commandStart = "start"
const commandHelp = "help"

func (b *Bot) handleCommand(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Я не знаю такой команды! 🌚")

	switch message.Command() {
	case commandStart:
		firstName := message.From.FirstName
		lastName := message.From.LastName
		greeting := fmt.Sprintf("Рад тебя приветствовать, %s %s! Я твой бот-репетитор по физике. Ты можешь выбрать раздел данной дисциплины для дальнейшей практики в виде решения моих задач. Выбери дальнейшие действия...", firstName, lastName)
		msg.Text = greeting
		msg.ReplyMarkup = b.startMenu
		msg.ParseMode = "markdown"
		_, err := b.bot.Send(msg)
		return err
	case commandHelp:
		msg.Text = "Нужна помощь? \n Свяжись с разработчиком! \n @Hummer4_YT к твоим услугам."
		_, err := b.bot.Send(msg)
		return err
	case "stat":
		score, rank := b.getUserStat(message.Chat.ID)
		msg.Text = fmt.Sprintf("Ваш рейтинг: %d баллов. Вы занимаете %d место среди всех пользователей.", score, rank)
		_, err := b.bot.Send(msg)
		return err
	default:
		_, err := b.bot.Send(msg)
		return err
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	b.mu.Lock()
	state, exists := b.userStates[message.Chat.ID]
	b.mu.Unlock()

	if exists && state.Next == "" {
		userAnswer, err := strconv.ParseInt(message.Text, 10, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, введите числовой ответ")
			b.sendMessage(msg)
			return
		}

		var result string
		if userAnswer == state.Answer {
			result = "Правильно! 🎉"
			b.updateUserScore(message.Chat.ID, state.Difficulty)
		} else {
			result = fmt.Sprintf("Неправильно! Правильный ответ: %d. Подумай еще или выбери другую задачу", state.Answer)
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, result)
		b.sendMessage(msg)

		msg = tgbotapi.NewMessage(message.Chat.ID, "Идем дальше?")
		msg.ReplyMarkup = b.nextTaskMenu
		b.sendMessage(msg)

		b.mu.Lock()
		state.Next = "next_task"
		b.userStates[message.Chat.ID] = state
		b.mu.Unlock()
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		b.bot.Send(msg)
	}
}

func (b *Bot) callbacks(update tgbotapi.Update) {
	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID
	firstName := update.CallbackQuery.From.FirstName
	lastName := update.CallbackQuery.From.LastName
	var text string

	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := b.bot.AnswerCallbackQuery(callback); err != nil {
		log.Println("Error sending callback response:", err)
	}

	switch data {
	case "choose_physics":
		text = "Какой раздел физики вы хотите выбрать?"
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.physicsMenu
		b.sendMessage(msg)
	case "dynamics", "mechanics", "electrostatics":
		b.mu.Lock()
		b.userStates[chatID] = UserState{Section: data}
		b.mu.Unlock()
		text = "Выберите уровень сложности"
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.difficultyMenu
		b.sendMessage(msg)
	case "beginner", "advanced", "expert":
		b.mu.Lock()
		state, exists := b.userStates[chatID]
		if exists {
			state.Difficulty = data
			b.userStates[chatID] = state
		}
		b.mu.Unlock()
		b.sendPhysicsTask(chatID, state.Section, state.Difficulty)
	case "next_task":
		b.mu.Lock()
		state, exists := b.userStates[chatID]
		b.mu.Unlock()
		if exists {
			b.sendPhysicsTask(chatID, state.Section, state.Difficulty)
		}
	case "hi":
		text = fmt.Sprintf("Hello %v %v", firstName, lastName)
	default:
		text = "Неизвестная команда"
	}
	if text != "" {
		msg := tgbotapi.NewMessage(chatID, text)
		b.sendMessage(msg)
	}
}

func (b *Bot) sendMessage(msg tgbotapi.Chattable) {
	if _, err := b.bot.Send(msg); err != nil {
		log.Panicf("Send message error: %v", err)
	}
}

func (b *Bot) sendPhysicsTask(chatID int64, section, difficulty string) {
	task, answer, err := getNextPhysicsTask(section, difficulty)
	if err != nil {
		log.Println("Error fetching task:", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при получении задачи")
		b.sendMessage(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, task)
	b.sendMessage(msg)

	b.mu.Lock()
	b.userStates[chatID] = UserState{Section: section, Difficulty: difficulty, Answer: answer}
	b.mu.Unlock()
}

func getNextPhysicsTask(section, difficulty string) (string, int64, error) {
	var task string
	var answer int64
	err := db.QueryRow("SELECT task, answer FROM physics_tasks WHERE section = $1 AND difficulty = $2 ORDER BY RANDOM() LIMIT 1", section, difficulty).Scan(&task, &answer)
	if err != nil {
		return "", 0, err
	}
	return task, answer, nil
}

func (b *Bot) updateUserScore(userID int64, difficulty string) {
	var score int
	switch difficulty {
	case "beginner":
		score = 1
	case "advanced":
		score = 2
	case "expert":
		score = 3
	default:
		score = 0
	}

	_, err := db.Exec("INSERT INTO user_scores (user_id, score) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET score = user_scores.score + $2", userID, score)
	if err != nil {
		log.Println("Error updating user score:", err)
	}
}

func (b *Bot) getUserStat(userID int64) (int, int) {
	var score int
	var rank int
	err := db.QueryRow("SELECT score FROM user_scores WHERE user_id = $1", userID).Scan(&score)
	if err != nil {
		log.Println("Error fetching user score:", err)
		return 0, 0
	}

	err = db.QueryRow("SELECT COUNT(*) FROM user_scores WHERE score > (SELECT score FROM user_scores WHERE user_id = $1)", userID).Scan(&rank)
	if err != nil {
		log.Println("Error fetching user rank:", err)
		return score, 0
	}

	rank += 1
	return score, rank
}
