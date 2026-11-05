import {InputTextarea} from "primereact/inputtextarea";
import {Button} from "primereact/button";
import {BsEmojiLaughing, BsSendFill} from "react-icons/bs";
import data from '@emoji-mart/data'
import Picker from '@emoji-mart/react'
import {useContext, useEffect, useRef, useState} from "react";
import "./MessageBox.css"
import {Card} from "primereact/card";
import avatarDemo from "../../assets/avatar.png"
import {MessageIn} from "./MessageIn.jsx";
import {MessageOut} from "./MessageOut.jsx";
import {UserContext} from "../../context/UserContext.jsx";
import {ChatContext} from "../../context/ChatContext.jsx";

export const MessageBox = props => {

  const {
    pubKeyFp,
    userId
  } = useContext(UserContext)

  const {
    chatId, chatGroup,
    chatMessages, sendMessage
  } = useContext(ChatContext)

  const [message, setMessage] = useState("")
  const [showEmoji, setShowEmoji] = useState(false)

	const showEmojiRef = useRef()
  const messagesEndRef = useRef(null);
  const messagesContainerRef = useRef(null);

  const sendMsg = (msg) => {
    if (msg !== "") {
      sendMessage(msg)
      setMessage("")
    }
  }

  const onMessageKeyDown = async (e) => {
    if (e.keyCode == 13 && e.shiftKey == false) {
      // Send message
      e.preventDefault();
      sendMsg(message)
    }
    else {
      setMessage(e.target.value)
    }
  }

  const scrollDown = (behavior="instant") => {
    messagesEndRef.current.scrollIntoView({ behavior: behavior });
  }

  const scrollDownIfBottom = () => {
    const messagesContainer = messagesContainerRef.current;
    const messagesEnd = messagesEndRef.current;
    const isScrolledToBottom =
      messagesContainer.scrollHeight - 50 - messagesContainer.clientHeight <=
      messagesContainer.scrollTop + 30;

    if (isScrolledToBottom) {
      // messagesEnd.scrollIntoView({ behavior: behavior });
      scrollDown("smooth")
    }
  }

	useEffect(() => {
		showEmojiRef.current = showEmoji
	}, [showEmoji])

	useEffect(() => {
		showEmojiRef.current = showEmoji
	}, [])

  useEffect(() => {
    setTimeout(()=>{scrollDown()}, 500)
  }, [chatId])

  useEffect(() => {
    scrollDownIfBottom()
  }, [chatMessages]);

  return(
    <div className='flex flex-col w-full h-full
    bg-gradient-to-l from-slate-500/70 to-slate-600/80'>

      {/*messages*/}
      <div className={"relative flex w-full h-[calc(100%-50px)]"} style={{overflow: "none"}}>
          <div
            className={"absolute sm:py-8 py-5 bottom-0 overflow-auto flex flex-col w-full max-h-full space-y-3"}
            ref={messagesContainerRef}
          >
            {chatMessages.map(m => {
              if (m.is_system || m.sender === 0 || m.sender === "system") {
                return (
                  <div key={m.id} className="w-full flex justify-center my-2">
                    <span className="text-xs text-slate-300 bg-slate-700/80 border border-slate-600/50 px-3 py-1 rounded-full font-light shadow-sm">
                      {m.msg}
                    </span>
                  </div>
                )
              }
              if (m.sender === userId) {
                return (
                  <MessageOut key={m.id} text={m.msg} messageData={m} />
                )
              }
              else {
                return(
                  <MessageIn key={m.id} text={m.msg} messageData={m} />
                )
              }
            })
            }
           <div ref={messagesEndRef} />
          </div>
      </div>

      {/*input*/}
      <div className={"relative flex flex-col w-full bg-white"}>
        <div className={"flex min-h-[50px] max-h-[50px]"}>
          <InputTextarea
            autoResize
            rows={2}
            className={"msg-input flex-grow my-1"}
            placeholder={"Write a message..."}
            value={message}
            onChange={e => { setMessage(e.target.value) }}
            onKeyDown={onMessageKeyDown}
          />
          <div className={"flex"}>
            {/*Emojis for desktop only*/}
            <Button
              className={"emoji-button hidden lg:flex my-auto p-button-text"}
              id={"emoji-button"}
              icon={<BsEmojiLaughing size={"25"} />}
              onClick={(e) => { setShowEmoji(!showEmojiRef.current) }}
            />
            <Button
							className={"my-auto mr-2 p-button-rounded"}
							icon={<BsSendFill />}
							disabled={(!message.length)}
							onClick={e => { sendMsg(message) }}
						/>
          </div>
        </div>
      </div>

      {showEmoji &&
        <div className={"absolute bottom-[95px] right-[45px] flex"}>
          <Picker
            className={"w-[300px]"}
            data={data}
            onEmojiSelect={e => { setMessage(message + e.native)  }}
            perLine={9}
            searchPosition={"none"}
            previewPosition={"none"}
            maxFrequentRows={0}
            onClickOutside={(e) => { if (e.target.id != "emoji-button") setShowEmoji(false) }}
          />
        </div>
      }
    </div>
  )
}
