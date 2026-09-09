import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import ReviewFlashcard from './ReviewFlashcard.vue'

function mockSpeech() {
  const spoken: SpeechSynthesisUtterance[] = []
  const cancel = vi.fn()
  const speak = vi.fn((utterance: SpeechSynthesisUtterance) =>
    spoken.push(utterance),
  )
  vi.stubGlobal('speechSynthesis', { speak, cancel })
  vi.stubGlobal(
    'SpeechSynthesisUtterance',
    class {
      text: string
      constructor(text: string) {
        this.text = text
      }
      lang = ''
      onend: (() => void) | null = null
      onerror: (() => void) | null = null
    },
  )
  return { spoken, speak, cancel }
}
const props = {
  autoPlayPronunciation: false,
  word: 'ambiguous',
  phonetic: '/æmˈbɪɡjuəs/',
  meaning: '模棱两可的',
  mode: 'recalling' as const,
  canCorrect: false,
  submitting: false,
  error: null,
}

describe('复习卡片发音', () => {
  it('默认首次播放一次，揭示和无关 props 更新不重播', async () => {
    const { spoken } = mockSpeech()
    const view = render(ReviewFlashcard, {
      props: { ...props, autoPlayPronunciation: undefined },
    })
    expect(spoken).toHaveLength(1)
    expect(spoken[0]).toMatchObject({ text: 'ambiguous', lang: 'en-US' })
    await view.rerender({ mode: 'revealed', canCorrect: true })
    await view.rerender({
      meaning: '新释义',
      phonetic: '',
      error: '重试',
      submitting: true,
    })
    expect(spoken).toHaveLength(1)
    view.unmount()
  })

  it('切词取消旧语音并自动播放，旧回调不能污染新状态', async () => {
    const { spoken, cancel } = mockSpeech()
    const view = render(ReviewFlashcard, {
      props: { ...props, autoPlayPronunciation: true },
    })
    const oldEnd = spoken[0].onend!
    const oldError = spoken[0].onerror!
    await view.rerender({ word: 'brisk' })
    expect(cancel).toHaveBeenCalledTimes(1)
    expect(spoken.map((item) => item.text)).toEqual(['ambiguous', 'brisk'])
    oldEnd.call(spoken[0], new Event('end') as SpeechSynthesisEvent)
    oldError.call(spoken[0], new Event('error') as SpeechSynthesisErrorEvent)
    expect(screen.getByRole('status')).toHaveTextContent(
      '正在播放 brisk 的发音',
    )
    view.unmount()
    expect(cancel).toHaveBeenCalledTimes(2)
  })

  it('关闭不自动播放，设置改变只影响下一词，批量更新读取最新设置', async () => {
    const { spoken, cancel } = mockSpeech()
    const view = render(ReviewFlashcard, { props })
    expect(spoken).toHaveLength(0)
    await view.rerender({ autoPlayPronunciation: true })
    expect(spoken).toHaveLength(0)
    await view.rerender({ word: 'brisk' })
    expect(spoken.map((item) => item.text)).toEqual(['brisk'])
    await view.rerender({ autoPlayPronunciation: false })
    expect(cancel).not.toHaveBeenCalled()
    await view.rerender({ word: 'cite' })
    expect(cancel).toHaveBeenCalledTimes(1)
    expect(spoken).toHaveLength(1)
    await view.rerender({ word: 'derive', autoPlayPronunciation: true })
    expect(spoken.map((item) => item.text)).toEqual(['brisk', 'derive'])
    view.unmount()
  })

  it('显示音标，空白音标不显示占位', async () => {
    const view = render(ReviewFlashcard, { props })
    expect(screen.getByText(props.phonetic)).toBeInTheDocument()
    await view.rerender({ phonetic: '  ' })
    expect(screen.queryByText(props.phonetic)).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: '播放 ambiguous 的发音' }),
    ).toBeEnabled()
  })

  it('两个阶段均可播放，重复播放取消旧请求，切词与卸载停止播放', async () => {
    const { spoken, cancel } = mockSpeech()
    const user = userEvent.setup()
    const view = render(ReviewFlashcard, { props })
    const play = screen.getByRole('button', { name: '播放 ambiguous 的发音' })
    await user.click(play)
    expect(spoken[0]).toMatchObject({ text: 'ambiguous', lang: 'en-US' })
    const staleError = spoken[0].onerror!
    await view.rerender({ mode: 'revealed' })
    await user.keyboard('{Enter}')
    expect(play).toHaveFocus()
    expect(spoken).toHaveLength(2)
    expect(cancel).toHaveBeenCalledTimes(1)
    await view.rerender({ word: 'brisk', phonetic: '' })
    expect(cancel).toHaveBeenCalledTimes(2)
    staleError.call(spoken[0], new Event('error') as SpeechSynthesisErrorEvent)
    expect(screen.getByRole('status')).toBeEmptyDOMElement()
    await user.click(screen.getByRole('button', { name: '播放 brisk 的发音' }))
    expect(spoken[2].text).toBe('brisk')
    expect(view.emitted().choose).toBeUndefined()
    expect(view.emitted().next).toBeUndefined()
    expect(view.emitted().correct).toBeUndefined()
    view.unmount()
    expect(cancel).toHaveBeenCalledTimes(3)
  })

  it('不支持时给出反馈，仍能作答', async () => {
    vi.stubGlobal('speechSynthesis', undefined)
    const user = userEvent.setup()
    const view = render(ReviewFlashcard, { props })
    await user.click(
      screen.getByRole('button', { name: '播放 ambiguous 的发音' }),
    )
    expect(screen.getByRole('status')).toHaveTextContent('当前浏览器不支持发音')
    await user.click(screen.getByRole('button', { name: /^认识$/ }))
    expect(view.emitted().choose).toEqual([['remembered']])
  })

  it('异步错误与同步异常可重试，完成后清除状态', async () => {
    const { spoken, speak } = mockSpeech()
    const user = userEvent.setup()
    render(ReviewFlashcard, { props })
    const play = screen.getByRole('button', { name: '播放 ambiguous 的发音' })
    await user.click(play)
    spoken[0].onerror!.call(
      spoken[0],
      new Event('error') as SpeechSynthesisErrorEvent,
    )
    await screen.findByText('发音播放失败，请重试或继续复习。')
    speak.mockImplementationOnce(() => {
      throw new Error('unavailable')
    })
    await user.click(play)
    expect(screen.getByRole('status')).toHaveTextContent('发音播放失败')
    await user.click(play)
    spoken[1].onend!.call(spoken[1], new Event('end') as SpeechSynthesisEvent)
    await vi.waitFor(() =>
      expect(screen.getByRole('status')).toBeEmptyDOMElement(),
    )
  })

  it('语音引擎无回调时超时降级', async () => {
    mockSpeech()
    vi.useFakeTimers()
    try {
      const view = render(ReviewFlashcard, { props })
      screen.getByRole('button', { name: '播放 ambiguous 的发音' }).click()
      await vi.advanceTimersByTimeAsync(15000)
      expect(screen.getByRole('status')).toHaveTextContent('发音播放超时')
      view.unmount()
    } finally {
      vi.useRealTimers()
    }
  })
})
