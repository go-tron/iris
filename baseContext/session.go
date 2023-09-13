package baseContext

//// Sessions stores gorilla sessions in Redis
//type Sessions struct {
//	CookieName string
//	// client to connect to redis
//	Client redis.UniversalClient
//	// default options to use when a new session is created
//	Options *sessions.Options
//	// key prefix with which the session will be stored
//	KeyPrefix string
//	// session serializer
//	Serializer SessionSerializer
//}
//
//type SessionsConfig struct {
//	Client     *redis.Client `json:"client"`
//	CookieName string        `json:"cookieName"`
//	MaxAge     time.Duration `json:"maxAge"`
//	HttpOnly   bool          `json:"httpOnly"`
//	KeyPrefix  string        `json:"keyPrefix"`
//}
//
//func NewSessionsWithConfig(c *config.Config, client *redis.Client) *Sessions {
//	return NewSessions(&SessionsConfig{
//		Client:     client,
//		CookieName: c.GetString("session.cookieName"),
//		MaxAge:     c.GetDuration("session.maxAge"),
//		HttpOnly:   c.GetBool("session.httpOnly"),
//		KeyPrefix:  c.GetString("application.name") + "-sess:",
//	})
//}
//
//// NewSessions returns a new Sessions with default configuration
//func NewSessions(c *SessionsConfig) *Sessions {
//	maxAge := int(c.MaxAge / time.Second)
//	rs := &Sessions{
//		Client:     c.Client,
//		CookieName: c.CookieName,
//		Options: &sessions.Options{
//			Path:     "/",
//			MaxAge:   maxAge,
//			HttpOnly: c.HttpOnly,
//		},
//		KeyPrefix:  c.KeyPrefix,
//		Serializer: JSONSerializer{},
//	}
//
//	if err := rs.Client.Ping(context.Background()).Err(); err != nil {
//		panic(err)
//	}
//	return rs
//}
//
//func (s *Sessions) GetDefault(r *http.Request) (*sessions.Session, error) {
//	return sessions.GetRegistry(r).Get(s, s.CookieName)
//}
//
//// Get returns a session for the given name after adding it to the registry.
////
//// See CookieStore.Get().
//func (s *Sessions) Get(r *http.Request, name string) (*sessions.Session, error) {
//	return sessions.GetRegistry(r).Get(s, name)
//}
//
//// New returns a session for the given name without adding it to the registry.
////
//// See CookieStore.New().
//func (s *Sessions) New(r *http.Request, name string) (*sessions.Session, error) {
//
//	session := sessions.NewSession(s, name)
//	opts := *s.Options
//	session.Options = &opts
//	session.IsNew = true
//
//	c, err := r.Cookie(name)
//	if err != nil {
//		return session, nil
//	}
//	session.ID = c.Value
//
//	err = s.load(session)
//	if err == nil {
//		session.IsNew = false
//	} else if err == redis.Nil {
//		err = nil // no data stored
//	}
//	return session, err
//}
//
//// Save adds a single session to the response.
////
//// If the Options.MaxAge of the session is <= 0 then the session file will be
//// deleted from the store path. With this process it enforces the properly
//// session cookie handling so no need to trust in the cookie management in the
//// web browser.
//func (s *Sessions) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
//	// Delete if max-age is < 0
//	if session.Options.MaxAge < 0 {
//		if err := s.erase(session); err != nil {
//			return err
//		}
//		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
//		return nil
//	}
//
//	if session.ID == "" {
//		// Because the ID is used in the filename, encode it to
//		// use alphanumeric characters only.
//		session.ID = strings.TrimRight(
//			base32.StdEncoding.EncodeToString(
//				securecookie.GenerateRandomKey(32)), "=")
//	}
//
//	if err := s.save(session); err != nil {
//		return err
//	}
//
//	http.SetCookie(w, sessions.NewCookie(session.Name(), session.ID, session.Options))
//	return nil
//}
//
//// save writes session in Redis
//func (s *Sessions) save(session *sessions.Session) error {
//
//	b, err := s.Serializer.Serialize(session)
//	if err != nil {
//		return err
//	}
//
//	age := session.Options.MaxAge
//	if age == 0 {
//		age = 60 * 10
//	}
//	return s.Client.Set(context.Background(), s.KeyPrefix+session.ID, b, time.Duration(age)*time.Second).Err()
//}
//
//// load reads session from Redis
//func (s *Sessions) load(session *sessions.Session) error {
//
//	cmd := s.Client.Get(context.Background(), s.KeyPrefix+session.ID)
//	if cmd.Err() != nil {
//		return cmd.Err()
//	}
//
//	b, err := cmd.Bytes()
//	if err != nil {
//		return err
//	}
//
//	return s.Serializer.Deserialize(b, session)
//}
//
//// delete deletes session in Redis
//func (s *Sessions) erase(session *sessions.Session) error {
//	return s.Client.Del(context.Background(), s.KeyPrefix+session.ID).Err()
//}
//
//// SessionSerializer provides an interface for serialize/deserialize a session
//type SessionSerializer interface {
//	Serialize(s *sessions.Session) ([]byte, error)
//	Deserialize(b []byte, s *sessions.Session) error
//}
//
//// JSONSerializer encode the session map to JSON.
//type JSONSerializer struct{}
//
//// Serialize to JSON. Will err if there are unmarshalable key values
//func (s JSONSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
//	m := make(map[string]interface{}, len(ss.Values))
//	for k, v := range ss.Values {
//		ks, ok := k.(string)
//		if !ok {
//			err := fmt.Errorf("Non-string key value, cannot serialize session to JSON: %v", k)
//			fmt.Printf("redistore.JSONSerializer.serialize() Error: %v", err)
//			return nil, err
//		}
//		m[ks] = v
//	}
//	return json.Marshal(m)
//}
//
//// Deserialize back to map[string]interface{}
//func (s JSONSerializer) Deserialize(d []byte, ss *sessions.Session) error {
//	m := make(map[string]interface{})
//	err := json.Unmarshal(d, &m)
//	if err != nil {
//		fmt.Printf("redistore.JSONSerializer.deserialize() Error: %v", err)
//		return err
//	}
//	for k, v := range m {
//		ss.Values[k] = v
//	}
//	return nil
//}
//
//// GobSerializer uses gob package to encode the session map
//type GobSerializer struct{}
//
//// Serialize using gob
//func (s GobSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
//	buf := new(bytes.Buffer)
//	enc := gob.NewEncoder(buf)
//	err := enc.Encode(ss.Values)
//	if err == nil {
//		return buf.Bytes(), nil
//	}
//	return nil, err
//}
//
//// Deserialize back to map[interface{}]interface{}
//func (s GobSerializer) Deserialize(d []byte, ss *sessions.Session) error {
//	dec := gob.NewDecoder(bytes.NewBuffer(d))
//	return dec.Decode(&ss.Values)
//}
//
//type Session struct {
//	*sessions.Session
//}
//
//func (s *Session) Set(key string, val interface{}) {
//	s.Values[key] = val
//	s.Values["needSave"] = 1
//}
//
//func (s *Session) Delete(key string) {
//	delete(s.Values, key)
//}
//
//func (s *Session) Increment(key string, n int) int {
//	newValue := s.GetInt(key)
//	newValue += n
//	s.Set(key, newValue)
//	return newValue
//}
//
//func (s *Session) Get(key string) interface{} {
//	return s.Values[key]
//}
//
//func (s *Session) GetString(key string) string {
//	return cast.ToString(s.Values[key])
//}
//
//func (s *Session) GetBool(key string) bool {
//	return cast.ToBool(s.Values[key])
//}
//
//func (s *Session) GetInt(key string) int {
//	return cast.ToInt(s.Values[key])
//}
//
//func (s *Session) GetInt32(key string) int32 {
//	return cast.ToInt32(s.Values[key])
//}
//
//func (s *Session) GetInt64(key string) int64 {
//	return cast.ToInt64(s.Values[key])
//}
//
//func (s *Session) GetUint(key string) uint {
//	return cast.ToUint(s.Values[key])
//}
//
//func (s *Session) GetUint32(key string) uint32 {
//	return cast.ToUint32(s.Values[key])
//}
//
//func (s *Session) GetUint64(key string) uint64 {
//	return cast.ToUint64(s.Values[key])
//}
//
//func (s *Session) GetFloat64(key string) float64 {
//	return cast.ToFloat64(s.Values[key])
//}
//
//func (s *Session) GetTime(key string) time.Time {
//	return cast.ToTime(s.Values[key])
//}
//
//func (s *Session) GetDuration(key string) time.Duration {
//	return cast.ToDuration(s.Values[key])
//}
//
//func (s *Session) GetIntSlice(key string) []int {
//	return cast.ToIntSlice(s.Values[key])
//}
//
//func (s *Session) GetStringSlice(key string) []string {
//	return cast.ToStringSlice(s.Values[key])
//}
//
//func (s *Session) GetStringMap(key string) map[string]interface{} {
//	return cast.ToStringMap(s.Values[key])
//}
//
//func (s *Session) GetStringMapString(key string) map[string]string {
//	return cast.ToStringMapString(s.Values[key])
//}
//
//func (s *Session) GetStringMapStringSlice(key string) map[string][]string {
//	return cast.ToStringMapStringSlice(s.Values[key])
//}
//
//const sessionContextKey = "session"
//
//func SessionHandler(ss *Sessions) iris.Handler {
//	return Handler(func(ctx *Context) {
//		if ss == nil {
//			panic("session undefined")
//		}
//		sess, _ := ss.GetDefault(ctx.Request())
//		if sess == nil {
//			return
//		}
//		ctx.Values().Set(sessionContextKey, &Session{Session: sess})
//		ctx.Next()
//		if needSave := sess.Values["needSave"]; needSave != nil {
//			delete(sess.Values, "needSave")
//			if err := sess.Save(ctx.Request(), ctx.ResponseWriter()); err != nil {
//				fmt.Println(err)
//			}
//		}
//	})
//}
//
//func (ctx *Context) GetSession() *Session {
//	if v := ctx.Values().Get(sessionContextKey); v != nil {
//		if sess, ok := v.(*Session); ok {
//			return sess
//		}
//	}
//	return nil
//}
//
//func (ctx *Context) SaveSession() (err error) {
//	sess := ctx.GetSession()
//	if sess == nil {
//		return nil
//	}
//	if needSave := sess.Values["needSave"]; needSave != nil {
//		delete(sess.Values, "needSave")
//	}
//	return sess.Save(ctx.Request(), ctx.ResponseWriter())
//}
