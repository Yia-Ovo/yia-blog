import { isLocale, localeNames, locales, useI18n } from '../i18n'

export default function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  return <label className="language-switcher">
    <span className="sr-only">{t('界面语言')}</span>
    <select value={locale} onChange={(event) => {
      if (isLocale(event.target.value)) setLocale(event.target.value)
    }}>
      {locales.map((value) => <option key={value} value={value} lang={value}>{localeNames[value]}</option>)}
    </select>
  </label>
}
