package main

import (
	"fmt"
	"math/rand"
	"os"

	"strconv"

	"time"

	"github.com/xiaonanln/goworld"
	"github.com/xiaonanln/goworld/common"
	"github.com/xiaonanln/goworld/consts"
	"github.com/xiaonanln/goworld/entity"
	"github.com/xiaonanln/goworld/gwlog"
	"github.com/xiaonanln/typeconv"
)

// Avatar entity which is the player itself
type Avatar struct {
	entity.Entity // Entity type should always inherit entity.Entity
}

func (a *Avatar) OnInit() {
}

func (a *Avatar) OnCreated() {
	a.Entity.OnCreated()

	a.setDefaultAttrs()

	gwlog.Info("Avatar %s on created: client=%s, mails=%d", a, a.GetClient(), a.Attrs.GetMapAttr("mails").Size())

	a.SetFilterProp("spaceKind", strconv.Itoa(a.GetInt("spaceKind")))
	a.SetFilterProp("level", strconv.Itoa(a.GetInt("level")))
	a.SetFilterProp("prof", strconv.Itoa(a.GetInt("prof")))

	//gwlog.Debug("Found OnlineService: %s", onlineServiceEid)
	a.CallService("OnlineService", "CheckIn", a.ID, a.Attrs.GetStr("name"), a.Attrs.GetInt("level"))

	a.AddTimer(time.Second, "PerSecondTick", 1, "")
}

func (a *Avatar) PerSecondTick(arg1 int, arg2 string) {
	fmt.Fprintf(os.Stderr, "!")
}

func (a *Avatar) setDefaultAttrs() {
	a.Attrs.SetDefault("name", "无名")
	a.Attrs.SetDefault("level", 1)
	a.Attrs.SetDefault("exp", 0)
	a.Attrs.SetDefault("prof", 1+rand.Intn(4))
	a.Attrs.SetDefault("spaceKind", 1+rand.Intn(100))
	a.Attrs.SetDefault("lastMailID", 0)
	a.Attrs.SetDefault("mails", goworld.MapAttr())
	a.Attrs.SetDefault("testListField", goworld.ListAttr())
}

func (a *Avatar) TestListField_Client() {
	testListField := a.GetListAttr("testListField")
	if testListField.Size() > 0 && rand.Float32() < 0.3333333333 {
		testListField.Pop()
	} else if testListField.Size() > 0 && rand.Float32() < 0.5 {
		testListField.Set(rand.Intn(testListField.Size()), rand.Intn(100))
	} else {
		testListField.Append(rand.Intn(100))
	}
	a.CallClient("OnTestListField", testListField.ToList())
}

func (a *Avatar) enterSpace(spaceKind int) {
	if a.Space.Kind == spaceKind {
		return
	}
	if consts.DEBUG_SPACES {
		gwlog.Info("%s enter space from %d => %d", a, a.Space.Kind, spaceKind)
	}
	a.CallService("SpaceService", "EnterSpace", a.ID, spaceKind)
}

func (a *Avatar) OnClientConnected() {
	//gwlog.Info("%s.OnClientConnected: current space = %s", a, a.Space)
	//a.Attrs.Set("exp", a.Attrs.GetInt("exp")+1)
	//a.Attrs.Set("testpop", 1)
	//v := a.Attrs.Pop("testpop")
	//gwlog.Info("Avatar pop testpop => %v", v)
	//
	//a.Attrs.Set("subattr", goworld.MapAttr())
	//subattr := a.Attrs.GetMapAttr("subattr")
	//subattr.Set("a", 1)
	//subattr.Set("b", 1)
	//subattr = a.Attrs.PopMapAttr("subattr")
	//a.Attrs.Set("subattr", subattr)

	a.SetFilterProp("online", "0")
	a.SetFilterProp("online", "1")
	a.enterSpace(a.GetInt("spaceKind"))
}

func (a *Avatar) OnClientDisconnected() {
	gwlog.Info("%s client disconnected", a)
	a.Destroy()
}

func (a *Avatar) EnterSpace_Client(kind int) {
	a.enterSpace(kind)
}

func (a *Avatar) DoEnterSpace(kind int, spaceID common.EntityID) {
	// let the avatar enter space with spaceID
	a.EnterSpace(spaceID, a.randomPosition())
}

func (a *Avatar) randomPosition() entity.Position {
	minCoord, maxCoord := -200, 200
	return entity.Position{
		X: entity.Coord(minCoord + rand.Intn(maxCoord-minCoord)),
		Y: entity.Coord(minCoord + rand.Intn(maxCoord-minCoord)),
		Z: entity.Coord(minCoord + rand.Intn(maxCoord-minCoord)),
	}
}

func (a *Avatar) OnEnterSpace() {
	if consts.DEBUG_SPACES {
		gwlog.Info("%s ENTER SPACE %s", a, a.Space)
	}
}

func (a *Avatar) GetSpaceID(callerID common.EntityID) {
	a.Call(callerID, "OnGetAvatarSpaceID", a.ID, a.Space.ID)
}

func (a *Avatar) OnDestroy() {
	a.CallService("OnlineService", "CheckOut", a.ID)
}

func (a *Avatar) SendMail_Client(targetID common.EntityID, mail MailData) {
	a.CallService("MailService", "SendMail", a.ID, a.GetStr("name"), targetID, mail)
}

func (a *Avatar) OnSendMail(ok bool) {
	a.CallClient("OnSendMail", ok)
}

// Avatar has received a mail, can query now
func (a *Avatar) NotifyReceiveMail() {
	//a.CallService("MailService", "GetMails", a.ID)
}

func (a *Avatar) GetMails_Client() {
	a.CallService("MailService", "GetMails", a.ID, a.GetInt("lastMailID"))
}

func (a *Avatar) OnGetMails(lastMailID int, mails []interface{}) {
	//gwlog.Info("%s.OnGetMails: lastMailID=%v/%v, mails=%v", a, a.GetInt("lastMailID"), lastMailID, mails)
	if lastMailID != a.GetInt("lastMailID") {
		gwlog.Warn("%s.OnGetMails: lastMailID mismatch: local=%v, return=%v", a, a.GetInt("lastMailID"), lastMailID)
		a.CallClient("OnGetMails", false)
		return
	}

	mailsAttr := a.Attrs.GetMapAttr("mails")
	for _, _item := range mails {
		item := _item.([]interface{})
		mailId := int(typeconv.Int(item[0]))
		if mailId <= a.GetInt("lastMailID") {
			gwlog.Panicf("mail ID should be increasing")
		}
		if mailsAttr.HasKey(strconv.Itoa(mailId)) {
			gwlog.Panicf("mail %d received multiple times", mailId)
		}
		mail := typeconv.String(item[1])
		mailsAttr.Set(strconv.Itoa(mailId), mail)
		a.Attrs.Set("lastMailID", mailId)
	}

	a.CallClient("OnGetMails", true)
}

func (a *Avatar) Say_Client(channel string, content string) {
	gwlog.Debug("Say @%s: %s", channel, content)
	if channel == "world" {
		a.CallFitleredClients("online", "1", "OnSay", a.ID, a.GetStr("name"), channel, content)
	} else if channel == "prof" {
		profStr := strconv.Itoa(a.GetInt("prof"))
		a.CallFitleredClients("prof", profStr, "OnSay", a.ID, a.GetStr("name"), channel, content)
	} else {
		gwlog.Panicf("%s.Say_Client: invalid channel: %s", a, channel)
	}
}

func (a *Avatar) Move_Client(pos entity.Position) {
	gwlog.Debug("Move from %s -> %s", a.GetPosition(), pos)
	a.SetPosition(pos)
}

//func (a *Avatar) getMailSenderInfo() map[string]interface{} {
//	return map[string]interface{}{
//		"ID":   a.ID,
//		"name": a.GetStr("name"),
//	}
//}
