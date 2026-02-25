import Logo from '@/assets/logo.svg?react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { motion, useScroll, useTransform, useInView, useSpring } from 'framer-motion';
import {
  ArrowRight,
  CheckCircle2,
  Github,
  Globe,
  Layers,
  Rocket,
  Shield,
  Zap,
} from 'lucide-react';
import { useRef } from 'react';
import { useNavigate } from '@tanstack/react-router';

// 滚动动画组件
function ScrollReveal({
  children,
  className = '',
  delay = 0,
  direction = 'up',
}: {
  children: React.ReactNode;
  className?: string;
  delay?: number;
  direction?: 'up' | 'down' | 'left' | 'right';
}) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: '-100px' });

  const directionMap = {
    up: { y: 60, x: 0 },
    down: { y: -60, x: 0 },
    left: { x: 60, y: 0 },
    right: { x: -60, y: 0 },
  };

  return (
    <motion.div
      ref={ref}
      className={className}
      initial={{
        opacity: 0,
        ...directionMap[direction],
      }}
      animate={
        isInView
          ? { opacity: 1, x: 0, y: 0 }
          : { opacity: 0, ...directionMap[direction] }
      }
      transition={{
        duration: 0.8,
        delay,
        ease: [0.25, 0.46, 0.45, 0.94],
      }}
    >
      {children}
    </motion.div>
  );
}

// 视差容器
function ParallaxSection({
  children,
  className = '',
  speed = 0.5,
}: {
  children: React.ReactNode;
  className?: string;
  speed?: number;
}) {
  const ref = useRef(null);
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ['start end', 'end start'],
  });

  const y = useTransform(scrollYProgress, [0, 1], [0, speed * 100]);
  const springY = useSpring(y, { stiffness: 100, damping: 30 });

  return (
    <motion.div ref={ref} style={{ y: springY }} className={className}>
      {children}
    </motion.div>
  );
}

// 数字递增动画
function CountUp({ target, suffix = '' }: { target: number; suffix?: string }) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true });
  const motionValue = useSpring(isInView ? target : 0, { stiffness: 100, damping: 30 });
  const display = useTransform(motionValue, (val: number) => Math.round(val));

  return (
    <motion.span ref={ref}>
      <motion.span>{display}</motion.span>
      {suffix}
    </motion.span>
  );
}

const features = [
  {
    icon: Zap,
    title: '极速开发',
    description: '基于 Vite 构建，毫秒级热更新，开发体验极致流畅',
    color: 'text-yellow-500',
    bg: 'bg-yellow-500/10',
  },
  {
    icon: Shield,
    title: '类型安全',
    description: 'TypeScript + TanStack Router，端到端类型安全',
    color: 'text-green-500',
    bg: 'bg-green-500/10',
  },
  {
    icon: Layers,
    title: '现代架构',
    description: 'React 19 + TanStack Query，组件化设计，易于扩展',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
  },
  {
    icon: Globe,
    title: '开箱即用',
    description: '内置认证、路由、状态管理，快速启动项目',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
  },
];

const techStack = [
  { name: 'React 19', category: '框架' },
  { name: 'TanStack Query', category: '数据获取' },
  { name: 'TanStack Router', category: '路由' },
  { name: 'Zustand', category: '状态管理' },
  { name: 'shadcn/ui', category: 'UI 组件' },
  { name: 'Tailwind v4', category: '样式' },
  { name: 'TypeScript', category: '类型' },
  { name: 'Vite', category: '构建' },
];

const stats = [
  { value: 100, suffix: '%', label: '类型覆盖' },
  { value: 50, suffix: 'ms', label: '热更新' },
  { value: 0, suffix: '依赖', label: '框架锁定' },
  { value: 100, suffix: '+', label: '组件' },
];

export default function LandingPage() {
  const navigate = useNavigate();
  const containerRef = useRef(null);
  const { scrollYProgress } = useScroll({ target: containerRef });
  const scaleX = useSpring(scrollYProgress, { stiffness: 100, damping: 30 });

  return (
    <div ref={containerRef} className="flex min-h-screen flex-col">
      {/* 进度条 */}
      <motion.div
        className="fixed left-0 right-0 top-0 z-[100] h-1 origin-left bg-primary"
        style={{ scaleX }}
      />

      {/* Header */}
      <motion.header
        initial={{ y: -100, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.6, ease: 'easeOut' }}
        className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-xl"
      >
        <div className="container flex h-16 items-center justify-between">
          <motion.div
            className="flex items-center gap-2"
            whileHover={{ scale: 1.05 }}
            transition={{ type: 'spring', stiffness: 400 }}
          >
            <Logo className="h-8 w-8" />
            <span className="text-xl font-bold">Project</span>
          </motion.div>
          <nav className="hidden items-center gap-8 md:flex">
            {['功能', '技术栈', '文档'].map((item, i) => (
              <motion.a
                key={item}
                href={i < 2 ? `#${['features', 'tech'][i]}` : 'https://github.com'}
                className="relative text-sm text-muted-foreground transition-colors hover:text-foreground"
                whileHover={{ y: -2 }}
                transition={{ type: 'spring', stiffness: 300 }}
              >
                {item}
              </motion.a>
            ))}
          </nav>
          <div className="flex items-center gap-4">
            <Button variant="ghost" onClick={() => navigate({ to: '/login' })}>
              登录
            </Button>
            <Button onClick={() => navigate({ to: '/login' })}>
              开始使用 <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </div>
      </motion.header>

      {/* Hero Section - 大气开场 */}
      <section className="relative flex min-h-[90vh] flex-col items-center justify-center overflow-hidden py-24">
        {/* 背景渐变 */}
        <div className="absolute inset-0 -z-10">
          <div className="absolute left-1/2 top-0 h-[800px] w-[800px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-gradient-to-b from-primary/20 to-transparent blur-3xl" />
          <div className="absolute bottom-0 left-0 h-[400px] w-[400px] rounded-full bg-gradient-to-tr from-violet-500/10 to-transparent blur-3xl" />
          <div className="absolute bottom-0 right-0 h-[400px] w-[400px] rounded-full bg-gradient-to-tl from-blue-500/10 to-transparent blur-3xl" />
        </div>

        <ScrollReveal>
          <motion.div
            className="rounded-full border bg-muted/50 px-4 py-1.5 text-sm text-muted-foreground"
            whileHover={{ scale: 1.05 }}
          >
            🚀 全新现代化模板，助力快速开发
          </motion.div>
        </ScrollReveal>

        <ScrollReveal delay={0.1}>
          <h1 className="mt-8 text-center text-5xl font-bold tracking-tighter sm:text-6xl md:text-7xl lg:text-8xl">
            构建下一代
            <br />
            <span className="bg-gradient-to-r from-blue-600 via-violet-600 to-purple-600 bg-clip-text text-transparent">
              Web 应用
            </span>
          </h1>
        </ScrollReveal>

        <ScrollReveal delay={0.2}>
          <p className="mx-auto mt-6 max-w-[700px] text-center text-lg text-muted-foreground md:text-xl">
            基于 React 19 + TanStack 生态的现代化项目模板，集成最佳实践，
            <br />
            让你专注于业务逻辑，而非基础设施。
          </p>
        </ScrollReveal>

        <ScrollReveal delay={0.3}>
          <div className="mt-10 flex gap-4">
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button size="lg" onClick={() => navigate({ to: '/login' })}>
                免费开始
              </Button>
            </motion.div>
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button size="lg" variant="outline">
                <Github className="mr-2 h-4 w-4" />
                GitHub
              </Button>
            </motion.div>
          </div>
        </ScrollReveal>

        {/* 代码预览 - 带浮动动画 */}
        <ParallaxSection className="mt-16 w-full max-w-3xl" speed={-0.2}>
          <ScrollReveal delay={0.4}>
            <motion.div
              className="overflow-hidden rounded-xl border bg-background/50 shadow-2xl backdrop-blur"
              whileHover={{ y: -5 }}
              transition={{ type: 'spring', stiffness: 300 }}
            >
              <div className="flex items-center gap-2 border-b px-4 py-3">
                <div className="h-3 w-3 rounded-full bg-red-500" />
                <div className="h-3 w-3 rounded-full bg-yellow-500" />
                <div className="h-3 w-3 rounded-full bg-green-500" />
                <span className="ml-2 text-xs text-muted-foreground">app.tsx</span>
              </div>
              <pre className="overflow-x-auto p-6 text-sm">
                <code className="text-muted-foreground">
                  {`import { useQuery } from '@tanstack/react-query'
import { useUserStore } from '@/store/user'

function App() {
  const { data } = useQuery({
    queryKey: ['users'],
    queryFn: fetchUsers,
  })

  return <UserList data={data} />
}`}
                </code>
              </pre>
            </motion.div>
          </ScrollReveal>
        </ParallaxSection>
      </section>

      {/* 数据统计 - 数字递增动画 */}
      <section className="border-y bg-muted/30 py-16">
        <div className="container">
          <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
            {stats.map((stat, i) => (
              <ScrollReveal key={stat.label} delay={i * 0.1}>
                <div className="text-center">
                  <div className="text-4xl font-bold text-primary">
                    <CountUp target={stat.value} suffix={stat.suffix} />
                  </div>
                  <div className="mt-2 text-sm text-muted-foreground">{stat.label}</div>
                </div>
              </ScrollReveal>
            ))}
          </div>
        </div>
      </section>

      {/* 功能展示 - 卡片依次出现 */}
      <section id="features" className="py-24">
        <div className="container">
          <ScrollReveal>
            <div className="mb-16 text-center">
              <motion.span
                className="text-sm font-medium text-primary"
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
              >
                核心功能
              </motion.span>
              <h2 className="mt-2 text-4xl font-bold">精心设计的每一个特性</h2>
              <p className="mt-4 text-muted-foreground">提升开发效率，让你专注于业务</p>
            </div>
          </ScrollReveal>

          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-4">
            {features.map((feature, i) => (
              <ScrollReveal key={feature.title} delay={i * 0.1} direction="up">
                <motion.div
                  whileHover={{ y: -8, scale: 1.02 }}
                  transition={{ type: 'spring', stiffness: 300 }}
                >
                  <Card className="h-full border-0 bg-background shadow-lg transition-shadow hover:shadow-xl">
                    <CardHeader>
                      <motion.div
                        className={`mb-4 flex h-14 w-14 items-center justify-center rounded-xl ${feature.bg}`}
                        whileHover={{ rotate: 10, scale: 1.1 }}
                        transition={{ type: 'spring' }}
                      >
                        <feature.icon className={`h-7 w-7 ${feature.color}`} />
                      </motion.div>
                      <CardTitle className="text-xl">{feature.title}</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <CardDescription className="text-base">{feature.description}</CardDescription>
                    </CardContent>
                  </Card>
                </motion.div>
              </ScrollReveal>
            ))}
          </div>
        </div>
      </section>

      {/* 技术栈 - 交错动画 */}
      <section id="tech" className="bg-muted/30 py-24">
        <div className="container">
          <ScrollReveal>
            <div className="mb-16 text-center">
              <span className="text-sm font-medium text-primary">技术栈</span>
              <h2 className="mt-2 text-4xl font-bold">采用业界最受欢迎的技术方案</h2>
            </div>
          </ScrollReveal>

          <div className="mx-auto grid max-w-4xl grid-cols-2 gap-6 md:grid-cols-4">
            {techStack.map((tech, i) => (
              <ScrollReveal key={tech.name} delay={i * 0.05}>
                <motion.div
                  whileHover={{ scale: 1.05, y: -5 }}
                  transition={{ type: 'spring', stiffness: 300 }}
                >
                  <Card className="cursor-pointer text-center transition-all hover:border-primary/50 hover:shadow-lg">
                    <CardContent className="pt-6">
                      <div className="text-lg font-semibold">{tech.name}</div>
                      <div className="mt-1 text-sm text-muted-foreground">{tech.category}</div>
                    </CardContent>
                  </Card>
                </motion.div>
              </ScrollReveal>
            ))}
          </div>
        </div>
      </section>

      {/* 大段展示 - 视差效果 */}
      <section className="relative overflow-hidden py-32">
        <div className="absolute inset-0 -z-10 bg-gradient-to-b from-background via-primary/5 to-background" />
        <ParallaxSection speed={0.3}>
          <div className="container text-center">
            <ScrollReveal>
              <motion.div
                className="inline-block rounded-2xl border bg-background/80 p-12 shadow-2xl backdrop-blur"
                whileHover={{ scale: 1.02 }}
                transition={{ type: 'spring', stiffness: 200 }}
              >
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 20, repeat: Infinity, ease: 'linear' }}
                  className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-primary/10"
                >
                  <Rocket className="h-10 w-10 text-primary" />
                </motion.div>
                <h2 className="text-3xl font-bold">快速启动你的项目</h2>
                <p className="mt-4 max-w-md text-muted-foreground">
                  从零到一，只需几分钟。内置完整的认证系统、路由管理和状态管理。
                </p>
                <motion.div className="mt-8" whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
                  <Button size="lg" onClick={() => navigate({ to: '/login' })}>
                    立即体验
                  </Button>
                </motion.div>
              </motion.div>
            </ScrollReveal>
          </div>
        </ParallaxSection>
      </section>

      {/* CTA Section */}
      <section className="bg-primary py-24">
        <div className="container text-center">
          <ScrollReveal>
            <h2 className="text-4xl font-bold text-primary-foreground">准备好了吗？</h2>
            <p className="mt-4 text-lg text-primary-foreground/80">
              立即开始构建你的下一个项目
            </p>
          </ScrollReveal>
          <ScrollReveal delay={0.2}>
            <motion.div
              className="mt-10"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Button
                size="lg"
                variant="secondary"
                onClick={() => navigate({ to: '/login' })}
              >
                <Rocket className="mr-2 h-4 w-4" />
                免费开始
              </Button>
            </motion.div>
          </ScrollReveal>
          <ScrollReveal delay={0.3}>
            <div className="mt-8 flex items-center justify-center gap-8 text-sm text-primary-foreground/60">
              {['免费开源', '无供应商锁定', '活跃社区'].map((item) => (
                <span key={item} className="flex items-center gap-1">
                  <CheckCircle2 className="h-4 w-4" />
                  {item}
                </span>
              ))}
            </div>
          </ScrollReveal>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t py-12">
        <div className="container">
          <div className="flex flex-col items-center justify-between gap-6 md:flex-row">
            <motion.div
              className="flex items-center gap-2"
              whileHover={{ scale: 1.05 }}
            >
              <Logo className="h-6 w-6" />
              <span className="font-semibold">Project</span>
            </motion.div>
            <nav className="flex gap-8">
              {['关于我们', '文档', 'GitHub', '联系我们'].map((item) => (
                <motion.a
                  key={item}
                  href="#"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                  whileHover={{ y: -2 }}
                >
                  {item}
                </motion.a>
              ))}
            </nav>
            <p className="text-sm text-muted-foreground">© 2026 Project. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
