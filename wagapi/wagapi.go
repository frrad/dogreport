package wagapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/zabawaba99/firego.v1"
)

type Client struct {
	httpClient *http.Client
	token      string
	ownerID    int64
}

type authResponse struct {
	Status string `json:"status"`
	Data   struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	} `json:"data"`
}

type jwtData struct {
	V    int64 `json:"v"`
	Data struct {
		Token   string `json:"token"`
		OwnerID int64  `json:"owner_id"`
		UID     string `json:"uid"`
	} `json:"d"`
	IAT int64 `json:"iat"`
}

type WalkType struct {
	AdditionalDogPrice int64   `json:"additional_dog_price"`
	CancelPrice        int64   `json:"cancel_price"`
	Description        string  `json:"description"`
	DescriptionShort   string  `json:"description_short"`
	ID                 int64   `json:"id"`
	Length             int64   `json:"length"`
	Name               string  `json:"name"`
	Price              float64 `json:"price"`
}

type NearbyWalkers struct {
	Expires int64   `json:"expires"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Walkers []struct {
		ID  string  `json:"id"`
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"walkers"`
}

type Review struct {
	BlockedWalker string `json:"blocked_walker"`
	Comment       string `json:"comment"`
	CreatedAt     string `json:"created_at"`
	Dog           struct {
		ID       string `json:"id"`
		ImageURL string `json:"image_url"`
		Name     string `json:"name"`
	} `json:"dog"`
	DogID              string `json:"dog_id"`
	ID                 string `json:"id"`
	IsAnonymous        string `json:"is_anonymous"`
	PreferredWalker    string `json:"preferred_walker"`
	Rating             string `json:"rating"`
	ReasonForBadReview string `json:"reason_for_bad_review"`
	UpdatedAt          string `json:"updated_at"`
	WalkID             string `json:"walk_id"`
	WalkerID           string `json:"walker_id"`
}

type Walker struct {
	CurrentLat float64 `json:"current_latitude"`
	CurrentLon float64 `json:"current_longitude"`
	Lat        float64 `json:"latitude"`
	Lon        float64 `json:"longitude"`

	Bio                string  `json:"bio"`
	FirstName          string  `json:"first_name"`
	ID                 int64   `json:"id"`
	Picture            string  `json:"picture"`
	WalkCompletedCount int32   `json:"walk_completed_count"`
	Gender             string  `json:"gender"`
	Rating             float32 `json:"rating"`
	Thumb              string  `json:"thumb"`
	Video              string  `json:"video"`
}

type Walk struct {
	Date     string  `json:"date"`
	Distance float64 `json:"distance"`
	Invoice  struct {
		Charges []struct {
			Amount      float64 `json:"amount"`
			Description string  `json:"description"`
		} `json:"charges"`
	} `json:"invoice"`
	IsPee         int64   `json:"is_pee"`
	IsPoo         int64   `json:"is_poo"`
	IsDoorLocked  int64   `json:"is_door_locked"`
	Note          string  `json:"note"`
	Payout        float64 `json:"payout"`
	PhotoURL      string  `json:"photo_url"`
	Tip           float64 `json:"tip"`
	Total         float64 `json:"total"`
	WalkCompleted string  `json:"walk_completed"`
	WalkEnd       string  `json:"walk_end"`
	WalkMap       string  `json:"walk_map"`
	WalkStart     string  `json:"walk_start"`
	WalkStarted   string  `json:"walk_started"`
	WalkerID      int64   `json:"walker_id"`
}

func NewClientWithToken(httpC *http.Client, token string) (*Client, error) {
	c := &Client{
		httpClient: httpC,
		token:      token,
	}
	err := c.setUserIdFromToken()
	return c, err
}

func NewClientWithUsernamePassword(httpC *http.Client, username, password string) (*Client, string, error) {
	token, err := getTokenUid(httpC, username, password)
	if err != nil {
		return nil, "", err
	}

	client, err := NewClientWithToken(httpC, token)
	if err != nil {
		return nil, "", err
	}
	return client, token, nil
}

func (c *Client) setUserIdFromToken() error {
	s := strings.Split(c.token, ".")
	if len(s) < 2 {
		return fmt.Errorf("Can't parse JWT %s", c.token)
	}
	data := s[1]
	sDec, _ := base64.StdEncoding.DecodeString(data)

	var structured jwtData
	if err := json.Unmarshal([]byte(string(sDec)+"}"), &structured); err != nil {
		return err
	}

	c.ownerID = structured.Data.OwnerID
	return nil
}

func (c *Client) ownerIDString() string {
	return strconv.FormatInt(c.ownerID, 10)
}

func (c *Client) QueryFirebase(suffix string, ans interface{}) {
	f := firego.New("https://wag-app.firebaseio.com/"+suffix, c.httpClient)
	f.Auth(c.token)

	if err := f.Value(ans); err != nil {
		log.Fatal(err)
	}
}

func (c *Client) LookupNearbyWalkers() NearbyWalkers {
	var ans NearbyWalkers
	c.QueryFirebase("walkers-nearby-owner/"+c.ownerIDString(), &ans)
	return ans
}

func (c *Client) LookupWalkTypes() []WalkType {
	var ans []WalkType
	c.QueryFirebase("walk-types", &ans)
	return ans
}

func (c *Client) LookupReviewsForWalker(walkerID string) []Review {
	var ans []Review
	c.QueryFirebase("walkers-reviews-by-walker/"+walkerID, &ans)
	return ans

}

func (c *Client) LookupReviewsForWalkerInt64(walkerID int64) []Review {
	walkerIDString := strconv.FormatInt(walkerID, 10)
	return c.LookupReviewsForWalker(walkerIDString)
}

func (c *Client) LookupPastWalks() map[string]Walk {
	var ans map[string]Walk
	c.QueryFirebase("walks-past-by-owner/"+c.ownerIDString(), &ans)
	return ans
}

func (c *Client) LookupWalker(walkerId string) Walker {
	var ans Walker
	c.QueryFirebase("walkers-profiles/"+walkerId, &ans)
	return ans
}

func (c *Client) LookupWalkerInt64(walkerId int64) Walker {
	walkerIDString := strconv.Itoa(int(walkerId))
	return c.LookupWalker(walkerIDString)
}

func (c *Client) LookupDog(dogId string) (ans interface{}) {
	c.QueryFirebase("dogs/"+dogId, &ans)
	return ans
}

func (c *Client) LookupOwner() interface{} {
	var ans interface{}
	c.QueryFirebase("owners/"+c.ownerIDString(), &ans)
	return ans
}

func getTokenUid(client *http.Client, username, password string) (string, error) {
	postData := []byte("type=owner&email=" + username + "&password=" + password)

	req, err := http.NewRequest("POST", "https://wag-middleman.herokuapp.com/login", bytes.NewReader(postData))
	if err != nil {
		panic("error 123123")
	}

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Accept-Language", "en-us")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Origin", "file://")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; XT1095 Build/MPES24.49-18-7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.2785.143 Crosswalk/23.53.589.4 Mobile Safari/537.3")

	resp, err := (*client).Do(req)
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom((*resp).Body)
	responseBody := buf.String()

	responseData := new(authResponse)
	if err := json.Unmarshal([]byte(responseBody), &responseData); err != nil {
		return "", err
	}

	if responseData.Status == "fail" {
		return "", fmt.Errorf("failed to acquire token with \nusername: %s\npassword: %s", username, password)
	}
	return responseData.Data.Token, nil
}
