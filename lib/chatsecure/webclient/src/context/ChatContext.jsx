import {createContext, useContext, useEffect, useRef, useState} from "react";
import {io} from "socket.io-client";
import {UserContext} from "./UserContext.jsx";
import {APIClient} from "../hooks/APIClient.jsx";
import { config } from "../utils/constants"

let socket;

export const ChatContext = createContext();

export const ChatProvider = (props) => {
	const {
	privKey, pubKey, pubKeyFp, token, userId, logout,
		encryptMessage
	} = useContext(UserContext)

	const [chatLoading, setChatLoading] = useState(true)
	const [miniLoading, setMiniLoading] = useState(true)
	const [userList, setUserList] = useState([])
	const [chatName, setChatName] = useState("Default")
	const [chatId, setChatId] = useState("default")  // Default group
	const [chatGroup, setChatGroup] = useState(true)  // Start showing group
	const [chatMessages, setChatMessages] = useState([])
	const [unreadMessages, setUnreadMessages] = useState([])

	const chatIdRef = useRef(chatId)
	const chatMessagesRef = useRef(chatMessages)
	const userListRef = useRef(userList)
	const unreadMessagesRef = useRef(unreadMessages)

	const api = APIClient(token)

	const getUserInfo = id => {
		return userListRef.current.filter(obj => obj.id === id)[0]
	}

	const updateUnreadMessages = (contactId, message) => {
		setUnreadMessages([...unreadMessagesRef.current, {id: contactId, msg: message}])
	}

	const cleanUnreadMessages = (contactId) => {
		const cleanedUnreadMessages = unreadMessagesRef.current.filter(obj => obj.id !== contactId)
		setUnreadMessages(cleanedUnreadMessages)
	}

	const updateUserList = ul => {
		const newUserList = ul.sort((a, b) => {
			return Number(b.active) - Number(a.active)
		})
		setUserList(newUserList)
	}

	function selectChat(id) {
		if (miniLoading)
			return

		if (id === "default") {
			// group
			setChatGroup(true)
			setChatName("Group")
		}
		else {
			// user
			setChatGroup(false)
			const user = getUserInfo(id)
			setChatName(user.name)
		}
		setChatId(id)
	}

	const sendMessage = async (msg) => {
		let keys = []
		if (chatId === "default") {
			// encrypt for all users
			keys = userListRef.current.map(u => u.pubkey)
		}
		else {
			// encrypt for single user
			keys = userListRef.current.filter(u => u.id === chatIdRef.current).map(u => u.pubkey)
		}
		keys.push(pubKey.armor())
		const encryptedMessage = await encryptMessage(msg, keys)
	const data = {
	  sender: userId,
	  recipient: chatId,
	  msg: encryptedMessage,
	  is_group: chatGroup,
	}
	socket.emit("msg", data)

	data.id = new Date().getTime()
	setChatMessages([...chatMessages, data])
	}

	function onMessage(value) {
		if (value.sender !== userId) {
			if (chatIdRef.current === "default" && value.recipient === "default") {
				setChatMessages([...chatMessagesRef.current, value])
			}
			if (chatIdRef.current !== "default"
				&& value.sender === chatIdRef.current
				&& value.recipient === userId) {
				setChatMessages([...chatMessagesRef.current, value])
			}

			if (value.sender !== chatIdRef.current && value.recipient === userId) {
				updateUnreadMessages(value.sender, value.msg)
			}
		}
	}

	function onUserConnected(data) {
		if (data.id !== userId) {
			// Check if user is already added
			const contactOnList = userListRef.current.find(obj => obj.id === data.id)
			if (contactOnList) {
				updateUserList([...userListRef.current.filter(obj => obj.id !== data.id), data])
			}
			else {
				updateUserList([...userListRef.current, data])
			}

			// Central system notification in general chat
			if (chatIdRef.current === "default") {
				const sysMsg = {
					id: "sys_" + Date.now() + "_" + Math.random(),
					sender: 0,
					recipient: "default",
					msg: `${data.name} connected`,
					is_system: true,
					ts: new Date().toISOString()
				}
				setChatMessages([...chatMessagesRef.current, sysMsg])
			}
		}
	}

	function onUserDisconnected (data) {
		const targetUser = userListRef.current.find(obj => obj.id === data.id)
		const userName = data.name || (targetUser ? targetUser.name : "Un usuario")
		updateUserList(
			userListRef.current.map(obj => {
				if (obj.id === data.id) {
					obj.active = false
				}
				return obj
			})
		)

		// Central system notification in general chat
		if (chatIdRef.current === "default") {
			const sysMsg = {
				id: "sys_" + Date.now() + "_" + Math.random(),
				sender: 0,
				recipient: "default",
				msg: `${userName} disconnected`,
				is_system: true,
				ts: new Date().toISOString()
			}
			setChatMessages([...chatMessagesRef.current, sysMsg])
		}
	}

	const initSocket = () => {
		socket = io(config.url.API_URL, {auth: {token: token}})
		socket.connect()
		initSocketEvents()
	}

	const initSocketEvents = () => {
		socket.on('connect', () => {
			// Refresh user list once the socket is established to catch any
			// user_connected events that may have fired during the handshake.
			api.getUserList().then(r => { if (r) updateUserList(r) })
		})
		socket.on('message', onMessage)
		socket.on('user_connected', onUserConnected)
		socket.on('user_disconnected', onUserDisconnected)
	}

	const initData = async () => {
    // await api.getGroupList().then(r => { console.log(r) })
    await api.getUserList().then(r => { updateUserList(r); })
		.catch(r => { logout() })
    await api.getGroupMessageList().then(r => { setChatMessages(r) })
    setChatLoading(false)
  }

	useEffect(() => {
	if (token !== null && token !== "") {
			initSocket()
			initData()
	}
	}, [token])

	useEffect(() => {
		setMiniLoading(true)
		chatIdRef.current = chatId
		cleanUnreadMessages(chatId)

		if (chatId !== "default") {
      api.getUserMessageList(chatId).then(setChatMessages)
    }
    else {
      api.getGroupMessageList().then(setChatMessages)
    }
	}, [chatId])

	useEffect(() => {
		chatMessagesRef.current = chatMessages
		setMiniLoading(false)
	}, [chatMessages])

	useEffect(() => {
		userListRef.current = userList
	}, [userList])

	useEffect(() => {
		unreadMessagesRef.current = unreadMessages
	}, [unreadMessages])


  return (
    <ChatContext.Provider
      value={{
		chatLoading, setChatLoading,
		miniLoading, setMiniLoading,
        userList, setUserList, updateUserList, getUserInfo,
        chatName, setChatName,
        chatId, setChatId,
        chatGroup, setChatGroup,
        chatMessages, setChatMessages,
        sendMessage, selectChat,
		unreadMessages, setUnreadMessages, unreadMessagesRef,
      }}
    >
      {props.children}
    </ChatContext.Provider>
  );
};
