import Logo from '@/assets/logo.svg?react';
import { Button } from '@/components/ui/button';
import { motion, useScroll, useTransform, useInView } from 'framer-motion';
import { ArrowRight, BarChart3, Bell, Clock, Shield, Smartphone, Zap } from 'lucide-react';
import { useRef } from 'react';
import { useNavigate } from '@tanstack/react-router';

// 滚动触发动画
function FadeIn({
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
  const isInView = useInView(ref, { once: true, margin: '-15%' });

  const dirMap = {
    up: { y: 80, x: 0 },
    down: { y: -80, x: 0 },
    left: { x: 80, y: 0 },
    right: { x: -80, y: 0 },
  };

  return (
    <motion.div
      ref={ref}
      className={className}
      initial={{ opacity: 0, ...dirMap[direction] }}
      animate={isInView ? { opacity: 1, x: 0, y: 0 } : { opacity: 0, ...dirMap[direction] }}
      transition={{ duration: 1, delay, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </motion.div>
  );
}

// 手机 Mockup
function PhoneMockup({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`relative ${className}`}>
      <div className="relative mx-auto w-[280px] overflow-hidden rounded-[40px] border-[8px] border-gray-900 bg-gray-900 shadow-2xl">
        {/* 刘海 */}
        <div className="absolute left-1/2 top-0 z-10 h-7 w-32 -translate-x-1/2 rounded-b-2xl bg-gray-900" />
        {/* 屏幕 */}
        <div className="relative aspect-[9/19.5] overflow-hidden bg-gradient-to-b from-blue-500 to-violet-600">
          {children}
        </div>
        {/* 底部指示条 */}
        <div className="absolute bottom-2 left-1/2 h-1 w-32 -translate-x-1/2 rounded-full bg-gray-600" />
      </div>
    </div>
  );
}

const features = [
  {
    icon: Clock,
    title: '时间追踪',
    description: '精确记录每一分钟，了解时间去向',
    color: 'from-blue-500 to-cyan-500',
  },
  {
    icon: BarChart3,
    title: '数据分析',
    description: '可视化报表，洞察工作效率',
    color: 'from-violet-500 to-purple-500',
  },
  {
    icon: Bell,
    title: '智能提醒',
    description: 'AI 驱动的提醒系统，不错过重要事项',
    color: 'from-orange-500 to-red-500',
  },
  {
    icon: Shield,
    title: '隐私安全',
    description: '端到端加密，数据完全掌控',
    color: 'from-green-500 to-emerald-500',
  },
];

export default function ProductPage() {
  const navigate = useNavigate();
  const containerRef = useRef(null);
  const { scrollYProgress } = useScroll({ target: containerRef });

  // 视差值
  const y1 = useTransform(scrollYProgress, [0, 1], [0, -100]);
  const opacity = useTransform(scrollYProgress, [0, 0.3], [1, 0]);

  return (
    <div ref={containerRef} className="bg-black text-white">
      {/* 固定导航 */}
      <motion.header
        className="fixed left-0 right-0 top-0 z-50"
        initial={{ y: -100 }}
        animate={{ y: 0 }}
        transition={{ duration: 0.6, delay: 0.2 }}
      >
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
          <div className="flex items-center gap-2">
            <Logo className="h-8 w-8" />
            <span className="text-lg font-semibold">Project</span>
          </div>
          <nav className="hidden items-center gap-8 md:flex">
            <a href="#features" className="text-sm text-gray-400 transition-colors hover:text-white">
              功能
            </a>
            <a href="#showcase" className="text-sm text-gray-400 transition-colors hover:text-white">
              展示
            </a>
            <a href="#" className="text-sm text-gray-400 transition-colors hover:text-white">
              定价
            </a>
          </nav>
          <Button variant="secondary" onClick={() => navigate({ to: '/login' })}>
            开始使用
          </Button>
        </div>
      </motion.header>

      {/* Hero Section - 全屏 */}
      <section className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6">
        {/* 背景光效 */}
        <div className="absolute inset-0">
          <div className="absolute left-1/2 top-1/2 h-[800px] w-[800px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-gradient-to-r from-blue-600/20 to-violet-600/20 blur-[120px]" />
        </div>

        <motion.div style={{ y: y1, opacity }} className="relative z-10 text-center">
          <FadeIn>
            <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-gray-800 bg-gray-900/50 px-4 py-2 text-sm text-gray-400">
              <Zap className="h-4 w-4 text-yellow-500" />
              全新发布 v2.0
            </div>
          </FadeIn>

          <FadeIn delay={0.1}>
            <h1 className="text-6xl font-bold tracking-tight sm:text-7xl md:text-8xl lg:text-9xl">
              <span className="bg-gradient-to-b from-white to-gray-400 bg-clip-text text-transparent">
                时间，
              </span>
              <br />
              <span className="bg-gradient-to-r from-blue-400 via-violet-400 to-purple-400 bg-clip-text text-transparent">
                尽在掌握
              </span>
            </h1>
          </FadeIn>

          <FadeIn delay={0.2}>
            <p className="mx-auto mt-8 max-w-2xl text-xl text-gray-400">
              重新定义时间管理。智能追踪、深度分析、高效协作，
              <br />
              让每一分钟都创造价值。
            </p>
          </FadeIn>

          <FadeIn delay={0.3}>
            <div className="mt-12 flex items-center justify-center gap-4">
              <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
                <Button size="lg" className="h-14 px-8 text-lg" onClick={() => navigate({ to: '/login' })}>
                  免费试用 <ArrowRight className="ml-2 h-5 w-5" />
                </Button>
              </motion.div>
              <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
                <Button size="lg" variant="outline" className="h-14 border-gray-700 px-8 text-lg">
                  <Smartphone className="mr-2 h-5 w-5" />
                  下载 App
                </Button>
              </motion.div>
            </div>
          </FadeIn>
        </motion.div>

        {/* 滚动提示 */}
        <motion.div
          className="absolute bottom-10 left-1/2 -translate-x-1/2"
          animate={{ y: [0, 10, 0] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <div className="flex flex-col items-center gap-2 text-sm text-gray-500">
            <span>向下滚动</span>
            <div className="h-10 w-6 rounded-full border-2 border-gray-700">
              <motion.div
                className="mx-auto mt-2 h-2 w-2 rounded-full bg-gray-500"
                animate={{ y: [0, 16, 0] }}
                transition={{ duration: 2, repeat: Infinity }}
              />
            </div>
          </div>
        </motion.div>
      </section>

      {/* 产品展示 - 带手机 mockup */}
      <section id="showcase" className="relative py-32">
        <div className="mx-auto max-w-7xl px-6">
          <FadeIn>
            <div className="mb-20 text-center">
              <span className="text-sm font-medium text-blue-400">产品展示</span>
              <h2 className="mt-4 text-5xl font-bold">简洁而强大</h2>
              <p className="mt-4 text-xl text-gray-400">直观的界面设计，让复杂变简单</p>
            </div>
          </FadeIn>

          <div className="grid items-center gap-16 lg:grid-cols-2">
            {/* 左侧内容 */}
            <FadeIn direction="left">
              <div className="space-y-8">
                <div>
                  <h3 className="text-3xl font-bold">实时追踪</h3>
                  <p className="mt-4 text-lg text-gray-400">
                    一键开始计时，自动分类任务。智能识别工作模式，
                    帮你了解每天的时间分配。
                  </p>
                </div>
                <div className="space-y-4">
                  {['自动追踪应用使用', '智能任务分类', '专注模式计时'].map((item, i) => (
                    <motion.div
                      key={item}
                      className="flex items-center gap-3"
                      initial={{ opacity: 0, x: -20 }}
                      whileInView={{ opacity: 1, x: 0 }}
                      viewport={{ once: true }}
                      transition={{ delay: i * 0.1 }}
                    >
                      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20">
                        <div className="h-2 w-2 rounded-full bg-blue-400" />
                      </div>
                      <span className="text-gray-300">{item}</span>
                    </motion.div>
                  ))}
                </div>
              </div>
            </FadeIn>

            {/* 右侧手机 */}
            <FadeIn direction="right">
              <PhoneMockup>
                <div className="flex h-full flex-col items-center justify-center p-6 text-center">
                  <div className="text-6xl font-bold">02:45</div>
                  <div className="mt-2 text-sm text-white/70">专注工作中...</div>
                  <div className="mt-8 flex gap-4">
                    <div className="h-16 w-16 rounded-full bg-white/20 backdrop-blur" />
                    <div className="h-16 w-16 rounded-full bg-white/30 backdrop-blur" />
                  </div>
                  <div className="mt-8 w-full space-y-3">
                    <div className="h-12 rounded-xl bg-white/10 backdrop-blur" />
                    <div className="h-12 rounded-xl bg-white/10 backdrop-blur" />
                    <div className="h-12 rounded-xl bg-white/10 backdrop-blur" />
                  </div>
                </div>
              </PhoneMockup>
            </FadeIn>
          </div>
        </div>
      </section>

      {/* 功能卡片 - 深色卡片风格 */}
      <section id="features" className="relative bg-gray-950 py-32">
        <div className="mx-auto max-w-7xl px-6">
          <FadeIn>
            <div className="mb-20 text-center">
              <span className="text-sm font-medium text-violet-400">核心功能</span>
              <h2 className="mt-4 text-5xl font-bold">为效率而生</h2>
            </div>
          </FadeIn>

          <div className="grid gap-6 md:grid-cols-2">
            {features.map((feature, i) => (
              <FadeIn key={feature.title} delay={i * 0.1}>
                <motion.div
                  className="group relative overflow-hidden rounded-2xl border border-gray-800 bg-gray-900/50 p-8 transition-colors hover:border-gray-700"
                  whileHover={{ y: -4 }}
                >
                  {/* 渐变背景 */}
                  <div
                    className={`absolute -right-20 -top-20 h-40 w-40 rounded-full bg-gradient-to-r ${feature.color} opacity-0 blur-[80px] transition-opacity group-hover:opacity-20`}
                  />
                  <div className="relative">
                    <div
                      className={`inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-r ${feature.color}`}
                    >
                      <feature.icon className="h-7 w-7 text-white" />
                    </div>
                    <h3 className="mt-6 text-2xl font-semibold">{feature.title}</h3>
                    <p className="mt-3 text-gray-400">{feature.description}</p>
                  </div>
                </motion.div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* 数据展示 */}
      <section className="relative py-32">
        <div className="mx-auto max-w-7xl px-6">
          <FadeIn>
            <div className="rounded-3xl bg-gradient-to-r from-blue-600 to-violet-600 p-16 text-center">
              <h2 className="text-4xl font-bold md:text-5xl">已被 10,000+ 团队信赖</h2>
              <p className="mt-4 text-xl text-white/80">加入他们，提升你的工作效率</p>
              <div className="mt-12 grid grid-cols-2 gap-8 md:grid-cols-4">
                {[
                  { value: '10K+', label: '活跃用户' },
                  { value: '50M+', label: '追踪小时' },
                  { value: '99.9%', label: '可用性' },
                  { value: '4.9', label: '用户评分' },
                ].map((stat) => (
                  <div key={stat.label}>
                    <div className="text-4xl font-bold">{stat.value}</div>
                    <div className="mt-2 text-white/70">{stat.label}</div>
                  </div>
                ))}
              </div>
            </div>
          </FadeIn>
        </div>
      </section>

      {/* CTA */}
      <section className="relative py-32">
        <div className="mx-auto max-w-4xl px-6 text-center">
          <FadeIn>
            <h2 className="text-5xl font-bold">准备好了吗？</h2>
            <p className="mt-6 text-xl text-gray-400">免费开始，无需信用卡</p>
            <motion.div className="mt-12" whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button size="lg" className="h-16 px-12 text-lg" onClick={() => navigate({ to: '/login' })}>
                立即开始 <ArrowRight className="ml-2 h-5 w-5" />
              </Button>
            </motion.div>
          </FadeIn>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-12">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6">
          <div className="flex items-center gap-2">
            <Logo className="h-6 w-6" />
            <span className="font-semibold">Project</span>
          </div>
          <p className="text-sm text-gray-500">© 2026 Project. All rights reserved.</p>
        </div>
      </footer>
    </div>
  );
}
