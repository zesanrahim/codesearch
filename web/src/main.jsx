import React from 'react'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import Landing from './App.jsx'
import AppShell from './routes/AppShell.jsx'
import Inbox from './routes/Inbox.jsx'
import Review from './routes/Review.jsx'
import './index.css'

const router = createBrowserRouter([
  { path: '/', element: <Landing /> },
  { path: '/how', element: <Landing /> },
  { path: '/demo', element: <Landing /> },
  {
    path: '/app',
    element: <AppShell />,
    children: [
      { index: true, element: <Inbox /> },
      { path: ':owner/:repo', element: <Inbox /> },
      { path: ':owner/:repo/pull/:number', element: <Review /> },
    ],
  },
])

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
)
