import avatarDemo from "../../assets/avatar.png";
import "./ContactList.css"
import Sha256 from "crypto-js/sha256.js";
import { Identicon } from "@polkadot/react-identicon";
import { useContext, useEffect, useState } from "react";
import { ChatContext } from "../../context/ChatContext.jsx";
import { MdOutlineKeyboardArrowDown, MdOutlineKeyboardArrowRight } from "react-icons/md";
import { TiGroup } from "react-icons/ti";
import { BsCircleFill } from "react-icons/bs";
import {UserContext} from "../../context/UserContext.jsx";

export const ContactList = props => {
	const { setVisible } = props

	const {
		decryptMessage
	} = useContext(UserContext)

	const {
		userList,
		chatId, selectChat,
		unreadMessagesRef,
		chatName } = useContext(ChatContext)


	const UserContact = props => {
		// Find unread messages
		const { contact, setVisible } = props
		const [lastMessage, setLastMessage] = useState("")

		const [unreadCount, setUnreadCount] = useState(0)
		useEffect(() => {
			const um = unreadMessagesRef.current.filter(obj => obj.id === contact.id)
			// console.log(um)
			setUnreadCount(um.length)
		}, [unreadMessagesRef.current])

		useEffect(() => {
			if (contact.msg && typeof contact.msg === "string") {
				decryptMessage(contact.msg).then(r => setLastMessage(r)).catch(() => {})
			}
		}, [])

		return (
			<div
			className={`grid grid-cols-[55px_1fr_auto] py-4 pl-10 pr-7 border-b border-slate-600 hover:bg-slate-600/50 cursor-pointer
				
				${(chatId === contact.id ? " bg-gradient-to-r from-slate-600 to-indigo-800 text-slate-100" : "text-slate-300")}`}
				onClick={e => { selectChat(contact.id); setVisible(false); }}
			>
				<div className={"w-full " + (contact.active === false ? "grayscale" : "")}>
					<div className='relative grid place-content-center w-full h-full object-cover rounded-full
					shadow-lg shadow-slate-800/20
					bg-gradient-to-b from-slate-700 to-slate-800'>
						<Identicon className={"m-auto"} size={33} value={String("0x" + Sha256(contact.fp))} theme={"substrate"} />
						{(Number(unreadCount) === 0) ?
							<div className={"absolute bottom-0 right-0 text-lime-400"}>
								<BsCircleFill size={12} />
							</div>
							:
							<div className={"absolute flex bottom-0 right-0 bg-red-500 w-4 h-4 rounded-full"}>
								<span className={"text-white text-[8px] m-auto"}>{unreadCount}</span>
							</div>
						}
					</div>
				</div>
				<div className='w-full ml-5'>
					<span className={"text-lg"}>{contact.name}</span>
					<span className={"text-base font-light line-clamp-1 opacity-50"}>{/*lastMessage*/}&nbsp;</span>
				</div>
				<div className="grid place-content-center">
					<MdOutlineKeyboardArrowRight className="text-3xl opacity-30" />
				</div>
			</div>
		)
	}

	return (
		<div className='flex flex-col w-full h-full overflow-auto bg-slate-700 p-0'>

			<div
				className={`grid grid-cols-[55px_1fr_auto] py-4 pl-5 pr-7 border-b border-slate-600 hover:bg-slate-600/50 cursor-pointer
				${(chatId === "default" ? " bg-gradient-to-r from-slate-600 to-indigo-800 text-slate-100" : "text-slate-300")}`}
				onClick={e => { selectChat("default"); setVisible(false); }}
			>
				<div className='w-full'>
					<div className='grid place-content-center w-full h-full object-cover rounded-full
					shadow-lg shadow-slate-800/20
					bg-gradient-to-b from-slate-400 to-slate-500'>
						<TiGroup className="text-4xl text-slate-100" />
					</div>
				</div>
				<div className='w-full ml-5'>
					<span className={"text-lg"}>General</span>
					<span className={"text-base font-light line-clamp-1 opacity-50"}>&nbsp;</span>
				</div>
				<div className="grid place-content-center">
					<MdOutlineKeyboardArrowDown className="text-3xl opacity-60" />
				</div>
			</div>
			{
				userList.map((contact) => (
					<UserContact setVisible={setVisible} key={contact.id} contact={contact} />
				))
			}
		</div>
	)
}
