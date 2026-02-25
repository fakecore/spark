import Logo from '@/assets/logo.svg?react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { ArrowRight, Github, LayoutDashboard, Sparkles } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';

export default function HomePage() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-screen flex-col">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur">
        <div className="container flex h-16 items-center justify-between">
          <div className="flex items-center gap-2">
            <Logo className="h-8 w-8" />
            <span className="text-xl font-bold">Project</span>
          </div>
          <div className="flex items-center gap-4">
            <Button variant="ghost" onClick={() => navigate({ to: '/login' })}>
              登录
            </Button>
            <Button onClick={() => navigate({ to: '/login' })}>
              开始使用
            </Button>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="container flex flex-1 flex-col items-center justify-center py-24 text-center">
        <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-primary/10">
          <Logo className="h-12 w-12" />
        </div>
        <h1 className="mt-8 text-4xl font-bold tracking-tight sm:text-5xl">
          Project Template
        </h1>
        <p className="mt-4 max-w-md text-lg text-muted-foreground">
          React 19 + TanStack + shadcn/ui 现代化项目模板
        </p>
        <div className="mt-10 flex flex-wrap justify-center gap-4">
          <Button size="lg" onClick={() => navigate({ to: '/login' })}>
            <LayoutDashboard className="mr-2 h-4 w-4" />
            进入系统
          </Button>
          <Button size="lg" variant="outline" onClick={() => navigate({ to: '/showcase' })}>
            <Sparkles className="mr-2 h-4 w-4" />
            苹果风格展示
          </Button>
          <Button size="lg" variant="outline" onClick={() => navigate({ to: '/product' })}>
            <ArrowRight className="mr-2 h-4 w-4" />
            产品展示页
          </Button>
        </div>

        {/* 技术栈 */}
        <div className="mt-24 grid w-full max-w-3xl grid-cols-2 gap-4 sm:grid-cols-4">
          {['React 19', 'TanStack', 'shadcn/ui', 'Tailwind v4'].map((tech) => (
            <Card key={tech} className="text-center">
              <CardContent className="pt-4 pb-4">
                <div className="text-sm font-medium">{tech}</div>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t py-6">
        <div className="container flex items-center justify-between text-sm text-muted-foreground">
          <span>© 2026 Project</span>
          <a href="https://github.com" className="flex items-center gap-1 hover:text-foreground">
            <Github className="h-4 w-4" />
            GitHub
          </a>
        </div>
      </footer>
    </div>
  );
}
