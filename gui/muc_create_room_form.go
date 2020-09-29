package gui

import (
	"github.com/coyim/coyim/coylog"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/coyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
	"github.com/golang-collections/collections/set"
)

type mucCreateRoomViewForm struct {
	ac      *connectedAccountsComponent
	isShown bool

	view             gtki.Box          `gtk-widget:"createRoomForm"`
	notificationArea gtki.Box          `gtk-widget:"notificationArea"`
	account          gtki.ComboBox     `gtk-widget:"accounts"`
	chatServices     gtki.ComboBoxText `gtk-widget:"chatServices"`
	chatServiceEntry gtki.Entry        `gtk-widget:"chatServicesEntry"`
	roomEntry        gtki.Entry        `gtk-widget:"roomNameEntry"`
	roomAutoJoin     gtki.CheckButton  `gtk-widget:"autoJoinCheckButton"`
	spinner          gtki.Spinner      `gtk-widget:"createRoomFormSpinner"`
	createButton     gtki.Button       `gtk-widget:"createRoomFormCreateButton"`

	errorBox     *errorNotification
	notification gtki.InfoBar

	previousUpdate          chan bool
	roomNameConflictList    *set.Set
	createRoom              func(*account, jid.Bare)
	onCheckFieldsConditions func(string, string, *account) bool

	log func(*account, jid.Bare) coylog.Logger
}

func (v *mucCreateRoomView) newCreateRoomForm() *mucCreateRoomViewForm {
	f := &mucCreateRoomViewForm{
		log:                  v.log,
		roomNameConflictList: set.New(),
	}

	f.initBuilder(v)
	f.initDefaults(v)

	return f
}

func (f *mucCreateRoomViewForm) initBuilder(v *mucCreateRoomView) {
	builder := newBuilder("MUCCreateRoomForm")
	panicOnDevError(builder.bindObjects(f))

	builder.ConnectSignals(map[string]interface{}{
		"on_cancel":          v.onCancel,
		"on_create_room":     f.onCreateRoom,
		"on_roomName_change": f.enableCreationIfConditionsAreMet,
		"on_roomAutoJoin_toggled": func() {
			v.updateAutoJoinValue(f.roomAutoJoin.GetActive())
		},
		"on_chatServiceEntry_change": f.enableCreationIfConditionsAreMet,
	})
}

func (f *mucCreateRoomViewForm) initDefaults(v *mucCreateRoomView) {
	f.errorBox = newErrorNotification(f.notificationArea)
	f.ac = v.u.createConnectedAccountsComponent(f.account, f, f.updateServicesBasedOnAccount, f.onNoAccountsConnected)
}

func (v *mucCreateRoomView) initCreateRoomForm() *mucCreateRoomViewForm {
	f := v.newCreateRoomForm()

	f.createRoom = func(ca *account, roomID jid.Bare) {
		errors := make(chan error)
		v.createRoom(ca, roomID, errors)
		go f.listenToCreateError(roomID, errors)
	}

	f.addCallbacks(v)

	return f
}

func (f *mucCreateRoomViewForm) listenToCreateError(roomID jid.Bare, errors chan error) {
	err := <-errors

	switch err {
	case errCreateRoomCheckIfExistsFails:
		doInUIThread(f.onCreateRoomCheckIfExistsFails)

	case errCreateRoomAlreadyExists:
		f.roomNameConflictList.Insert(roomID.String())
		doInUIThread(f.onCreateRoomAlreadyExists)

	case errCreateRoomFailed:
		doInUIThread(func() {
			f.onCreateRoomFailed(err)
		})
	}
}

func (f *mucCreateRoomViewForm) onCreateRoomCheckIfExistsFails() {
	f.errorBox.ShowMessage(i18n.Local("Couldn't connect to the service, please verify that it exists or try again later."))
	f.hideSpinner()
	f.enableFields()
}

func (f *mucCreateRoomViewForm) onCreateRoomAlreadyExists() {
	f.errorBox.ShowMessage(i18n.Local("That room already exists, try again with a different name."))
	f.hideSpinner()
	f.enableFields()
	f.createButton.SetSensitive(false)
}

func (f *mucCreateRoomViewForm) onCreateRoomFailed(err error) {
	displayErr, ok := supportedCreateMUCErrors[err]
	if ok {
		f.errorBox.ShowMessage(displayErr)
		return
	}
	f.errorBox.ShowMessage(i18n.Local("Could not create the room."))
}

func (f *mucCreateRoomViewForm) addCallbacks(v *mucCreateRoomView) {
	v.onAutoJoin.add(func() {
		f.onAutoJoinChange(v.autoJoin)
	})

	v.onDestroy.add(f.destroy)
}

func (f *mucCreateRoomViewForm) showCreateForm(v *mucCreateRoomView) {
	v.success.reset()
	v.container.Remove(v.success.view)
	f.reset()
	v.container.Add(f.view)
	f.isShown = true
}

func (f *mucCreateRoomViewForm) onAutoJoinChange(v bool) {
	if v {
		f.createButton.SetProperty("label", i18n.Local("Create Room & Join"))
	} else {
		f.createButton.SetProperty("label", i18n.Local("Create Room"))
	}
}

func (f *mucCreateRoomViewForm) onCreateRoom() {
	f.clearErrors()

	roomName, _ := f.roomEntry.GetText()
	local := jid.NewLocal(roomName)
	if !local.Valid() {
		f.log(nil, nil).WithField("local", roomName).Error("Trying to create a room with an invalid local")
		f.notifyOnError(i18n.Local("You must provide a valid room name."))
		return
	}

	chatService, _ := f.chatServiceEntry.GetText()
	domain := jid.NewDomain(chatService)
	if !domain.Valid() {
		f.log(nil, nil).WithField("domain", chatService).Error("Trying to create a room with an invalid domain")
		f.notifyOnError(i18n.Local("You must provide a valid service name."))
		return
	}

	roomID := jid.NewBare(local, domain)

	ca := f.ac.currentAccount()
	if ca == nil {
		f.log(nil, roomID).Error("No account was selected to create the room")
		f.notifyOnError(i18n.Local("No account is selected, please select one account from the list or connect to one."))
		return
	}

	f.beforeCreatingTheRoom()

	go f.createRoom(ca, roomID)
}

func (f *mucCreateRoomViewForm) beforeCreatingTheRoom() {
	f.showSpinner()
	f.disableFields()
}

func (f *mucCreateRoomViewForm) destroy() {
	f.isShown = false
	f.ac.onDestroy()
}

func (f *mucCreateRoomViewForm) notifyOnError(err string) {
	if f.notification != nil {
		f.notificationArea.Remove(f.notification)
	}
	f.errorBox.ShowMessage(err)
}

func (f *mucCreateRoomViewForm) clearErrors() {
	if f.isShown {
		f.errorBox.Hide()
	}
}

func (f *mucCreateRoomViewForm) clearFields() {
	f.roomEntry.SetText("")
	f.enableCreationIfConditionsAreMet()
}

func (f *mucCreateRoomViewForm) reset() {
	f.spinner.Stop()
	f.enableFields()
	f.clearFields()
}

func (f *mucCreateRoomViewForm) setFieldsSensitive(v bool) {
	f.createButton.SetSensitive(v)
	f.account.SetSensitive(v)
	f.roomEntry.SetSensitive(v)
	f.chatServices.SetSensitive(v)
	f.roomAutoJoin.SetSensitive(v)
}

func (f *mucCreateRoomViewForm) disableFields() {
	f.setFieldsSensitive(false)
	f.ac.disableAccountInput()
}

func (f *mucCreateRoomViewForm) enableFields() {
	f.setFieldsSensitive(true)
	f.ac.enableAccountInput()
}

func (f *mucCreateRoomViewForm) updateServicesBasedOnAccount(ca *account) {
	doInUIThread(func() {
		f.clearErrors()
		f.enableCreationIfConditionsAreMet()
	})
	go f.updateChatServicesBasedOnAccount(ca)
}

func (f *mucCreateRoomViewForm) onNoAccountsConnected() {
	doInUIThread(func() {
		f.enableCreationIfConditionsAreMet()
		f.chatServices.RemoveAll()
	})
}

func (f *mucCreateRoomViewForm) enableCreationIfConditionsAreMet() {
	// Let the connected accounts component show any errors if it have one
	if len(f.ac.accounts) > 0 {
		f.clearErrors()
	}

	roomName, _ := f.roomEntry.GetText()
	chatService, _ := f.chatServiceEntry.GetText()
	currentAccount := f.ac.currentAccount()

	ok := len(roomName) != 0 && len(chatService) != 0 && currentAccount != nil
	if ok {
		roomID := jid.NewBare(jid.NewLocal(roomName), jid.NewDomain(chatService))
		if roomID.Valid() && f.roomNameConflictList.Has(roomID.String()) {
			f.errorBox.ShowMessage(i18n.Local("That room already exists, try again with a different name."))
			ok = false
		}
	}

	f.createButton.SetSensitive(ok)
}

func (f *mucCreateRoomViewForm) updateChatServicesBasedOnAccount(ca *account) {
	if f.previousUpdate != nil {
		f.previousUpdate <- true
	}

	f.previousUpdate = make(chan bool)

	csc, ec, endEarly := ca.session.GetChatServices(jid.ParseDomain(ca.Account()))

	go f.updateChatServices(ca, csc, ec, endEarly)
}

func (f *mucCreateRoomViewForm) updateChatServices(ca *account, csc <-chan jid.Domain, ec <-chan error, endEarly func()) {
	hadAny := false
	ts := make(chan string)

	doInUIThread(func() {
		t, _ := f.chatServiceEntry.GetText()
		ts <- t
		f.chatServices.RemoveAll()
		f.showSpinner()
	})

	typedService := <-ts

	defer func() {
		// We call this in a anonymous function because we want
		// always the latest value of hadAny and we use it as a
		// global variable in this context
		f.onUpdateChatServicesFinished(hadAny, typedService)
	}()

	for {
		select {
		case <-f.previousUpdate:
			doInUIThread(f.chatServices.RemoveAll)
			endEarly()
			return
		case err, _ := <-ec:
			if err != nil {
				f.log(ca, nil).WithError(err).Error("Something went wrong trying to get chat services")
			}
			return
		case cs, ok := <-csc:
			if !ok {
				return
			}

			hadAny = true
			doInUIThread(func() {
				f.chatServices.AppendText(cs.String())
			})
		}
	}
}

func (f *mucCreateRoomViewForm) onUpdateChatServicesFinished(hadAny bool, typedService string) {
	if hadAny && typedService == "" {
		doInUIThread(func() {
			f.chatServices.SetActive(0)
		})
	}

	doInUIThread(f.hideSpinner)

	f.previousUpdate = nil
}

func (f *mucCreateRoomViewForm) showSpinner() {
	f.spinner.Start()
	f.spinner.Show()
}

func (f *mucCreateRoomViewForm) hideSpinner() {
	f.spinner.Stop()
	f.spinner.Hide()
}

func setEnabled(w gtki.Widget, enable bool) {
	w.SetSensitive(enable)
}
