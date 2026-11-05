import {useContext, useEffect, useState} from "react";
import {ChatContext} from "../../context/ChatContext.jsx";
import Sha256 from "crypto-js/sha256.js";
import {Identicon} from "@polkadot/react-identicon";
import {UserContext} from "../../context/UserContext.jsx";

export const MessageIn = props => {
	const {
		decryptMessage
	} = useContext(UserContext)
  const {
    userList, getUserInfo,
  } = useContext(ChatContext)

  const {
    messageData
  } = props

	const [message, setMessage] = useState("")
	const [userInfo, setUserInfo] = useState({
		active: true,
		fp: "",
		id: "",
		last_update: "",
		name: "",
		pubkey: "",
	})
  const [fp, setFp] = useState("")

  const getFp = () => {
    const user_id = messageData.sender
    const user = userList.filter(u => u.id === user_id)
    if (user && user.length > 0 && user[0].fp) {
      return user[0].fp
    }
    return ""
  }

  useEffect(() => {
    if (messageData) {
      const userFp = getFp()
      const fp = userFp ? String("0x" + Sha256(userFp)) : ""
      setFp(fp)
      const ui = getUserInfo(messageData.sender)
      if (ui) {
        setUserInfo(ui)
      }
    }

    if (props.text && props.text.includes("-----BEGIN PGP MESSAGE-----")) {
      decryptMessage(props.text).then(r => setMessage(r)).catch(() => {})
    } else {
      setMessage(props.text || "")
    }
  }, [messageData, userList])

  return (
    <div className={'grid gap-4 w-full sm:grid-cols-[84px_2fr_1fr] grid-cols-[64px_1fr] place-items-start ' + (userInfo.active===false&&"grayscale hover:grayscale-0")}>
      <div className='sm:ml-[40px] ml-[20px]'>
        <Identicon className={"mx-auto"} size={44} value={ fp } theme={"substrate"} />
      </div>
      <div className="bg-slate-700 w-fit
      px-5 pt-2 pb-2.5 mr-5 sm:mr-0
      rounded-tr-2xl rounded-bl-2xl rounded-br-2xl
      text-base text-slate-300 font-light">
				<div className={"grid"}>
					<span className={"text-xs text-blue-500 right-0"}>{userInfo.name}</span>
				</div>
				<p>{message}</p>
      </div>
      <div className="hidden sm:block"></div>
    </div>
  )
}
