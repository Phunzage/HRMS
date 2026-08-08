import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  message: string
}

// 全局错误边界：任何子组件渲染异常时展示可操作错误页，避免整棵树卸载导致黑屏、无法交互
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, message: '' }

  static getDerivedStateFromError(error: unknown): State {
    return { hasError: true, message: error instanceof Error ? error.message : String(error) }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // 错误已记录到 console，便于后端排查
    console.error('页面渲染异常:', error, info.componentStack)
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-base-200 p-6">
          <div className="card w-full max-w-md bg-base-100 shadow-xl">
            <div className="card-body items-center text-center">
              <h1 className="card-title text-3xl font-bold text-error">页面出错了</h1>
              <p className="mt-2 text-base-content/70">抱歉，页面加载时发生了意外错误。</p>
              {this.state.message && (
                <div className="mt-3 w-full overflow-x-auto rounded-lg bg-base-200 p-3 text-left text-xs text-base-content/70">
                  {this.state.message}
                </div>
              )}
              <div className="mt-4 flex flex-wrap justify-center gap-2">
                <button type="button" className="btn btn-primary" onClick={this.handleReload}>
                  重新加载
                </button>
                <a href="/" className="btn btn-outline">
                  返回首页
                </a>
                <button type="button" className="btn btn-ghost" onClick={() => window.history.back()}>
                  返回上一页
                </button>
              </div>
              <p className="mt-6 text-xs text-base-content/40">错误已记录，如反复出现请联系管理员</p>
            </div>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
