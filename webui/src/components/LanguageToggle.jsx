import { useI18n } from '../i18n'
import { useState, useRef, useEffect } from 'react'

export default function LanguageToggle({ className = '' }) {
    const { lang, setLang, t } = useI18n()
    const [isOpen, setIsOpen] = useState(false)
    const dropdownRef = useRef(null)

    const languages = [
        { code: 'vi', label: t('language.vietnamese') },
        { code: 'en', label: t('language.english') }
    ]

    const currentLang = languages.find(l => l.code === lang)

    useEffect(() => {
        const handleClickOutside = (event) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
                setIsOpen(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    return (
        <div className={`relative ${className}`} ref={dropdownRef}>
            <button
                type="button"
                onClick={() => setIsOpen(!isOpen)}
                className="text-xs font-semibold px-2 py-1 rounded-md border border-border bg-secondary/50 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                title={t('language.label')}
            >
                {currentLang?.label || 'Language'}
            </button>
            {isOpen && (
                <div className="absolute right-0 mt-1 w-32 rounded-md border border-border bg-popover shadow-lg z-50">
                    {languages.map((l) => (
                        <button
                            key={l.code}
                            type="button"
                            onClick={() => {
                                setLang(l.code)
                                setIsOpen(false)
                            }}
                            className={`w-full text-left px-3 py-2 text-xs hover:bg-accent transition-colors ${
                                lang === l.code ? 'bg-accent font-semibold' : ''
                            }`}
                        >
                            {l.label}
                        </button>
                    ))}
                </div>
            )}
        </div>
    )
}
