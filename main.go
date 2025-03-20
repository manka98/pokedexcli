package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/manka98/pokedexcli/internal/pokecache"
)

func cleanInput(text string) []string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\s-]`)
	textWithoutSpecialChars := re.ReplaceAllString(text, "")
	textLowerCase := strings.ToLower(textWithoutSpecialChars)

	words := strings.Fields(textLowerCase)
	return words
}

type cliCommand struct {
	name        string
	description string
	callback    func(config *Config, cache *pokecache.Cache) error
}

func commandExit(config *Config, cache *pokecache.Cache) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}
func helpMessage(config *Config, cache *pokecache.Cache) error {
	// fmt.Println("Helping you")
	return nil
}

type Location struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type Response struct {
	Count    int        `json:"count"`
	Next     string     `json:"next"`
	Previous *string    `json:"previous"`
	Results  []Location `json:"results"`
}

type Config struct {
	Next     string
	Previous *string
}

type Version struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EncounterMethod struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type VersionDetails struct {
	Rate    int     `json:"rate"`
	Version Version `json:"version"`
}

type EncounterMethodRate struct {
	EncounterMethod EncounterMethod  `json:"encounter_method"`
	VersionDetails  []VersionDetails `json:"version_details"`
}
type Language struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Name struct {
	Language Language `json:"language"`
	Name     string   `json:"name"`
}

type ConditionValue struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EncounterDetails struct {
	Chance          int              `json:"chance"`
	ConditionValues []ConditionValue `json:"condition_values"`
	MaxLevel        int              `json:"max_level"`
	MinLevel        int              `json:"min_level"`
	Method          EncounterMethod  `json:"method"`
}

type PokemonVersionDetail struct {
	EncounterDetails []EncounterDetails `json:"encounter_details"`
	MaxChance        int                `json:"max_chance"`
	Version          Version            `json:"version"`
}

type Pokemon struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PokemonEncounter struct {
	Pokemon        Pokemon                `json:"pokemon"`
	VersionDetails []PokemonVersionDetail `json:"version_details"`
}

type PokedexLocation struct {
	EncounterMethodRates []EncounterMethodRate `json:"encounter_method_rates"`
	GameIndex            int                   `json:"game_index"`
	ID                   int                   `json:"id"`
	Location             Location              `json:"location"`
	Name                 string                `json:"name"`
	Names                []Name                `json:"names"`
	PokemonEncounters    []PokemonEncounter    `json:"pokemon_encounters"`
}

func commandMap(config *Config, cache *pokecache.Cache) error {

	if data, found := cache.Get(config.Next); found {
		var listLocation Response
		if err := json.Unmarshal(data, &listLocation); err != nil {
			return err
		}
		printLocations(listLocation)
		config.Next = listLocation.Next
		config.Previous = listLocation.Previous
		return nil
	}
	resp, err := http.Get(config.Next)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("network failed")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cache.Add(config.Next, data)

	var listLocation Response
	if err = json.Unmarshal(data, &listLocation); err != nil {
		return err
	}

	config.Next = listLocation.Next
	config.Previous = listLocation.Previous

	printLocations(listLocation)

	return nil
}

func printLocations(listLocation Response) {
	for _, v := range listLocation.Results {
		fmt.Println(v.Name)
	}
}

func commandMapb(config *Config, cache *pokecache.Cache) error {
	resp, err := http.Get(*config.Previous)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("network failed")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var listLocation Response

	if err = json.Unmarshal(data, &listLocation); err != nil {
		return err
	}
	for _, v := range listLocation.Results {
		fmt.Println(v.Name)

	}
	config.Next = listLocation.Next
	if listLocation.Previous == nil {
		fmt.Println("you're on the first page")
		return nil
	} else {
		config.Previous = listLocation.Previous
	}

	return nil
}

func exploreCommand(cache *pokecache.Cache, input []string) error {
	if len(input) < 2 {
		return errors.New("error - invalid name")
	}
	url := fmt.Sprint("https://pokeapi.co/api/v2/location-area/", input[1], "/")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("api request failed with status code")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var locationData PokedexLocation
	if err := json.Unmarshal(data, &locationData); err != nil {
		return err
	}

	for _, encounter := range locationData.PokemonEncounters {
		fmt.Println(encounter.Pokemon.Name)
	}

	return nil
}

type Stat struct {
	BaseStat int    `json:"base_stat"`
	Name     string `json:"name"`
}

type Type struct {
	Name string `json:"name"`
}

type PokemonData struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []Stat `json:"stats"`
	Types          []Type `json:"types"`
}
type Pokedex struct {
	CaughtPokemon map[string]PokemonData
}

var userPokedex = Pokedex{CaughtPokemon: make(map[string]PokemonData)}

func catchCommand(cache *pokecache.Cache, input []string) error {
	if len(input) < 2 {
		return errors.New("error - invalid name")
	}

	pokemonName := strings.ToLower(input[1])
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

	if _, exists := userPokedex.CaughtPokemon[pokemonName]; exists {
		fmt.Printf("%s is already in your Pokedex!\n", pokemonName)
		return nil
	}

	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("API request failed with status code")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		Name           string `json:"name"`
		BaseExperience int    `json:"base_experience"`
		Height         int    `json:"height"`
		Weight         int    `json:"weight"`
		Stats          []struct {
			BaseStat int `json:"base_stat"`
			Stat     struct {
				Name string `json:"name"`
			} `json:"stat"`
		} `json:"stats"`
		Types []struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"types"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}

	pokemon := PokemonData{
		Name:           response.Name,
		BaseExperience: response.BaseExperience,
		Height:         response.Height,
		Weight:         response.Weight,
	}

	for _, stat := range response.Stats {
		pokemon.Stats = append(pokemon.Stats, Stat{
			BaseStat: stat.BaseStat,
			Name:     stat.Stat.Name,
		})
	}

	for _, typ := range response.Types {
		pokemon.Types = append(pokemon.Types, Type{
			Name: typ.Type.Name,
		})
	}

	rand.Seed(time.Now().UnixNano())
	catchChance := rand.Intn(100) + 1
	threshold := 100 - pokemon.BaseExperience/2

	if catchChance > threshold {
		fmt.Printf("%s was caught!\n", pokemon.Name)
		userPokedex.CaughtPokemon[pokemonName] = pokemon
	} else {
		fmt.Printf("%s escaped!\n", pokemon.Name)
	}

	return nil
}

func inspectCommand(input []string) error {
	if len(input) < 2 {
		return errors.New("error - invalid name")
	}

	pokemonName := strings.ToLower(input[1])
	pokemon, exists := userPokedex.CaughtPokemon[pokemonName]

	if !exists {
		fmt.Println("You have not caught that Pokemon")
		return nil
	}

	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  - %s: %d\n", stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, ptype := range pokemon.Types {
		fmt.Printf("  - %s\n", ptype.Name)
	}

	return nil
}

func pokedexcommand(config *Config, cache *pokecache.Cache) error {
	fmt.Println("Your Pokedex: ")
	for _, v := range userPokedex.CaughtPokemon {
		fmt.Sprint("-", v.Name, "\n")

	}
	return nil
}

var commands = map[string]cliCommand{
	"exit": {
		name:        "exit",
		description: "Exit the Pokedex",
		callback:    commandExit,
	},
	"help": {
		name:        "help",
		description: "Helping",
		callback:    helpMessage,
	},
	"map": {
		name:        "map",
		description: "List of map",
		callback:    commandMap,
	},
	"mapb": {
		name:        "mapb",
		description: "Prev list of map",
		callback:    commandMapb,
	},
	"pokedex": {
		name:        "pokedex",
		description: "Check your pokedex",
		callback:    pokedexcommand,
	},
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	config := &Config{Next: "https://pokeapi.co/api/v2/location-area/", Previous: nil}
	cache := pokecache.NewCache(10)
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage: \n\nhelp: Displays a help message\nexit: Exit the Pokedex\nmap: Find a location\nCatch: Catch pokemon!")
	for {
		fmt.Println("Pokedex >")
		scanner.Scan()
		input := scanner.Text()
		cleanedInput := cleanInput(input)
		if len(cleanedInput) > 0 {
			if strings.Contains(cleanedInput[0], "explore") {
				exploreCommand(cache, cleanedInput)

			} else if strings.Contains(cleanedInput[0], "catch") {
				catchCommand(cache, cleanedInput)

			} else if strings.Contains(cleanedInput[0], "inspect") {
				inspectCommand(cleanedInput)

			} else {
				if cmd, found := commands[cleanedInput[0]]; found {
					err := cmd.callback(config, cache)
					if err != nil {
						fmt.Println("error")
					}
				}
			}
		}

	}

}
