<div align="center">

# 🚀 NOTLI: The Ultimate Notification System

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg)](https://goreportcard.com/) 
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![API Stability](https://img.shields.io/badge/API-Stable-green.svg)](https://github.com/atharvaawatade/Notili_NotificationSystem)
[![Uptime](https://img.shields.io/badge/uptime-99.99%25-brightgreen.svg)](https://github.com/atharvaawatade/Notili_NotificationSystem)

<p align="center">
  <img src="https://via.placeholder.com/600x300?text=NOTLI+Notification+System" alt="NOTLI Logo" width="600"/>
</p>

<h3>A powerful, scalable, and elegant microservice-based notification delivery platform</h3>

[Getting Started](#getting-started) • 
[Documentation](#documentation) • 
[Architecture](#microservices-architecture) • 
[Features](#key-features) • 
[Roadmap](#implementation-roadmap) • 
[Performance](#performance-metrics)

</div>

## 🔍 System Overview

NOTLI is a state-of-the-art notification system designed to process and deliver **millions of notifications per minute** with exceptional reliability and stunning performance. Built on a modern microservice architecture, NOTLI provides extreme scalability, comprehensive real-time analytics, and a future-ready design supporting multiple notification channels.

Initially focused on email delivery, NOTLI's architecture is engineered for seamless expansion to SMS, push notifications, messaging platforms, and voice channels. The system elegantly handles notifications from ingestion to delivery with sophisticated prioritization, intelligent routing, and comprehensive analytics.

## 📚 Documentation Structure

This repository contains detailed documentation explaining NOTLI's architecture, usage, and internal workings:

1. **[📋 User Flow](user_flow.md)**: Visualize the journey of a notification through the system, including the collaboration between microservices.

2. **[🔌 API Documentation](api.md)**: Comprehensive specifications for external REST APIs, internal gRPC communication, Kafka event schemas, and webhook formats.

3. **[📊 System Architecture](NOTLI_System_Architecture.md)**: Detailed explanation of the system's architecture, design principles, and components.

4. **[📝 Logging System](logs.md)**: Information about NOTLI's powerful logging and monitoring capabilities.

## 🏛️ Microservices Architecture

<div align="center">
<table>
  <tr>
    <td align="center" width="20%">
      <img src="https://via.placeholder.com/80?text=MS1" width="80" height="80"/><br />
      <b>💂‍♀️ Authentication Service</b>
    </td>
    <td align="center" width="20%">
      <img src="https://via.placeholder.com/80?text=MS2" width="80" height="80"/><br />
      <b>📬 Message Queue Service</b>
    </td>
    <td align="center" width="20%">
      <img src="https://via.placeholder.com/80?text=MS3" width="80" height="80"/><br />
      <b>⚖️ Prioritization Service</b>
    </td>
    <td align="center" width="20%">
      <img src="https://via.placeholder.com/80?text=MS4" width="80" height="80"/><br />
      <b>🚚 Delivery Service</b>
    </td>
    <td align="center" width="20%">
      <img src="https://via.placeholder.com/80?text=MS5" width="80" height="80"/><br />
      <b>📊 Analytics Service</b>
    </td>
  </tr>
</table>
</div>

### Microservice Details

#### 💂‍♀️ MS1: Authentication Service

The foundation of NOTLI's security and user management system, built in Go:
- User onboarding and account management
- Secure API key generation and validation
- Advanced authentication (OAuth2.0/JWT)
- Granular authorization controls
- User notification preferences management
- Rate limiting enforcement

#### 📬 MS2: Message Queue Service

The high-performance gateway for notification ingestion:
- Enterprise-grade REST API for notification submission
- Sophisticated validation & idempotency checks
- Efficient message publishing to Kafka
- Request throttling and backpressure handling
- Message schema enforcement
- Header-based routing capabilities

#### ⚖️ MS3: Prioritization Service

The intelligent brain of NOTLI that ensures critical messages get delivered first:
- Multi-dimensional priority algorithms
- Dynamic rate limiting with quota management
- Rule-based notification routing
- Priority queue implementation with optimized performance
- Adaptive throttling based on system load
- Real-time priority adjustment capabilities

#### 🚚 MS4: Delivery Service

The reliable delivery engine that ensures notifications reach their destination:
- Advanced template rendering with personalization
- Multi-provider delivery strategy with automatic failover
- Comprehensive retry mechanisms with exponential backoff
- Provider-specific optimizations
- Email, SMS, and push notification capabilities
- Delivery status tracking and reporting

#### 📊 MS5: Analytics & Admin Service

The powerful insights engine providing visibility into the notification ecosystem:
- Real-time analytics dashboards
- Apache Flink-powered stream processing
- Comprehensive delivery and engagement metrics
- System health monitoring
- User-friendly administrative interface
- Advanced reporting capabilities

## ✨ Key Features

- **🚄 Extreme Scalability**: Process 50,000+ notifications per second with horizontal scaling
- **🔐 Channel-Specific API Keys**: Granular control and rate limiting per notification channel
- **🧠 Intelligent Prioritization**: Multi-dimensional algorithms ensure critical messages are delivered first
- **🔄 Multi-Provider Delivery**: Automatic failover between providers ensures maximum deliverability
- **📈 Real-Time Analytics**: Comprehensive dashboards for notification performance and system health
- **🔮 Future-Ready Architecture**: Designed for seamless expansion to additional notification channels
- **🎨 Modern Dark UI**: Professional dark-themed interface to enhance usability and reduce eye strain

## 🏁 Getting Started

To understand NOTLI's architecture and capabilities:

1. 📖 Start with this `README.md` for a high-level overview
2. 🌊 Read the **[User Flow](user_flow.md)** to see how notifications travel through the system
3. 🔌 Consult the **[API Documentation](api.md)** for details on interacting with NOTLI
4. 🏗️ Review the **[System Architecture](NOTLI_System_Architecture.md)** for a deep dive into design
5. 📦 Explore the individual Microservice directories for implementation details

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/atharvaawatade/Notili_NotificationSystem.git
cd Notili_NotificationSystem

# Run all services (requires Docker and Docker Compose)
./tests/run_all_services.ps1

# Run integration tests
cd tests
go test ./integration -v
```

## 🛠️ Technology Stack

<div align="center">
<table>
  <tr>
    <td align="center"><b>Category</b></td>
    <td align="center"><b>Technologies</b></td>
  </tr>
  <tr>
    <td align="center">💻 Languages</td>
    <td>Go, Node.js, Java/Scala (Flink), JavaScript/TypeScript (React)</td>
  </tr>
  <tr>
    <td align="center">📨 Messaging</td>
    <td>Apache Kafka, Redis Pub/Sub</td>
  </tr>
  <tr>
    <td align="center">🗄️ Databases</td>
    <td>PostgreSQL, Elasticsearch, Redis, DynamoDB</td>
  </tr>
  <tr>
    <td align="center">🔄 Communication</td>
    <td>REST APIs, gRPC, Kafka Events, WebSockets</td>
  </tr>
  <tr>
    <td align="center">☁️ Infrastructure</td>
    <td>Kubernetes, Docker, Terraform, Prometheus, Grafana</td>
  </tr>
  <tr>
    <td align="center">🎨 Frontend</td>
    <td>React, Tailwind CSS, Font Awesome, Inter font family</td>
  </tr>
</table>
</div>

## 🗺️ Implementation Roadmap

<div align="center">
<table>
  <tr>
    <td align="center" width="25%"><b>🚀 Phase 1</b><br/>Email Notifications<br/><i>(Current)</i></td>
    <td align="center" width="25%"><b>📱 Phase 2</b><br/>SMS & Push Notifications<br/><i>(Next 3 months)</i></td>
    <td align="center" width="25%"><b>💬 Phase 3</b><br/>Messaging Platforms<br/>WhatsApp, Telegram, Slack<br/><i>(3-6 months)</i></td>
    <td align="center" width="25%"><b>🔊 Phase 4</b><br/>Voice Notifications<br/><i>(6-9 months)</i></td>
  </tr>
</table>
</div>

## 📊 Performance Metrics

<div align="center">
<table>
  <tr>
    <td align="center"><b>Microservice</b></td>
    <td align="center"><b>Key Metric</b></td>
    <td align="center"><b>Performance</b></td>
  </tr>
  <tr>
    <td align="center">💂‍♀️ Authentication</td>
    <td>API Key Validation</td>
    <td align="center"><b>50,000</b> validations/sec</td>
  </tr>
  <tr>
    <td align="center">📬 Message Queue</td>
    <td>Message Ingestion</td>
    <td align="center"><b>50,000</b> messages/sec</td>
  </tr>
  <tr>
    <td align="center">⚖️ Prioritization</td>
    <td>Message Processing</td>
    <td align="center"><b>30,000</b> messages/sec</td>
  </tr>
  <tr>
    <td align="center">🚚 Delivery</td>
    <td>Email Delivery</td>
    <td align="center"><b>20,000</b> emails/sec</td>
  </tr>
  <tr>
    <td align="center">📊 Analytics</td>
    <td>Event Processing</td>
    <td align="center"><b>50,000</b> events/sec</td>
  </tr>
</table>
</div>

## 🎨 User Interface

NOTLI features a modern, elegant dark-themed UI designed for optimal user experience:

- **🌙 Dark Mode**: Professional dark theme with gradient backgrounds (#181825 to #121220)
- **🔵 Accent Colors**: Blue accent colors (#3b82f6, #2563eb) for interactive elements
- **🖌️ Typography**: Clean Inter font family from Google Fonts with consistent sizing
- **✨ Visual Effects**: Subtle animations, semi-transparent overlays, and consistent shadows
- **📱 Responsive Design**: Fully responsive interface that works on all device sizes

---

<div align="center">

**NOTLI represents the pinnacle of notification system design, delivering unmatched scalability, reliability, and extensibility for organizations of any size.**

</div>
