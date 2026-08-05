export const resources = {
  en: {
    translation: {
      brand: {
        name: "Infrastructure Monitor",
        environment: "Private VPS dashboard",
      },
      navigation: {
        primary: "Primary navigation",
        overview: "Overview",
        cpu: "CPU",
        memory: "Memory",
        filesystems: "Filesystems",
        docker: "Docker",
        events: "Events",
        audit: "Audit",
        backups: "Backups",
      },
      language: {
        label: "Interface language",
        english: "English",
        french: "French",
      },
      actions: {
        skipToContent: "Skip to main content",
        returnToOverview: "Return to overview",
      },
      status: {
        healthy: "Healthy",
        warning: "Warning",
        critical: "Critical",
        unknown: "Unknown",
      },
      state: {
        loading: {
          title: "Loading",
          message: "Waiting for the latest monitoring data.",
        },
        empty: {
          title: "No data yet",
          message: "This area will populate when its feature is implemented.",
        },
        stale: {
          title: "Data is stale",
          message: "The last known values are older than expected.",
        },
        unavailable: {
          title: "Unavailable",
          message: "This information is not available right now.",
        },
        error: {
          title: "Something went wrong",
          message: "The request could not be completed.",
        },
      },
      overview: {
        eyebrow: "System overview",
        title: "Your VPS at a glance",
        introduction:
          "The shared dashboard shell is ready. Collectors will add live host and Docker data in later tasks.",
        healthTitle: "Overall health",
        healthMessage:
          "No monitoring sample has been collected by this version yet.",
        sectionsTitle: "Monitoring areas",
      },
      pages: {
        eyebrow: "Monitoring area",
        notReady: "Feature foundation ready",
        cpu: {
          title: "CPU",
          description:
            "Overall CPU, logical vCPUs, load averages, thresholds, and history will appear here.",
        },
        memory: {
          title: "Memory",
          description:
            "RAM, swap, pressure, history, and container ranking will appear here.",
        },
        filesystems: {
          title: "Filesystems",
          description:
            "Mounted filesystems, capacity, permissions, I/O, and history will appear here.",
        },
        docker: {
          title: "Docker",
          description:
            "Container state, health, resources, ports, and details will appear here.",
        },
        events: {
          title: "Events",
          description: "Warning and Critical event history will appear here.",
        },
        audit: {
          title: "Audit",
          description: "Administrative action history will appear here.",
        },
        backups: {
          title: "Backups",
          description:
            "Backup status, manual creation, and confirmed recovery will appear here.",
        },
      },
      notFound: {
        eyebrow: "404",
        title: "Page not found",
        message: "The requested dashboard page does not exist.",
      },
      footer: {
        access: "Tailnet access only",
        timezone: "Times display in Africa/Tunis",
      },
    },
  },
  fr: {
    translation: {
      brand: {
        name: "Moniteur d’infrastructure",
        environment: "Tableau de bord VPS privé",
      },
      navigation: {
        primary: "Navigation principale",
        overview: "Vue d’ensemble",
        cpu: "Processeur",
        memory: "Mémoire",
        filesystems: "Systèmes de fichiers",
        docker: "Docker",
        events: "Événements",
        audit: "Audit",
        backups: "Sauvegardes",
      },
      language: {
        label: "Langue de l’interface",
        english: "Anglais",
        french: "Français",
      },
      actions: {
        skipToContent: "Aller au contenu principal",
        returnToOverview: "Retourner à la vue d’ensemble",
      },
      status: {
        healthy: "Sain",
        warning: "Avertissement",
        critical: "Critique",
        unknown: "Inconnu",
      },
      state: {
        loading: {
          title: "Chargement",
          message: "En attente des dernières données de surveillance.",
        },
        empty: {
          title: "Aucune donnée",
          message:
            "Cette zone sera remplie lorsque sa fonctionnalité sera implémentée.",
        },
        stale: {
          title: "Données anciennes",
          message:
            "Les dernières valeurs connues sont plus anciennes que prévu.",
        },
        unavailable: {
          title: "Indisponible",
          message: "Ces informations ne sont pas disponibles actuellement.",
        },
        error: {
          title: "Une erreur s’est produite",
          message: "La demande n’a pas pu être effectuée.",
        },
      },
      overview: {
        eyebrow: "Vue du système",
        title: "Votre VPS en un coup d’œil",
        introduction:
          "La structure commune du tableau de bord est prête. Les collecteurs ajouteront les données réelles de l’hôte et de Docker dans les prochaines tâches.",
        healthTitle: "État général",
        healthMessage:
          "Aucun échantillon de surveillance n’a encore été collecté par cette version.",
        sectionsTitle: "Zones de surveillance",
      },
      pages: {
        eyebrow: "Zone de surveillance",
        notReady: "Base de la fonctionnalité prête",
        cpu: {
          title: "Processeur",
          description:
            "L’utilisation globale, les processeurs logiques, la charge, les seuils et l’historique apparaîtront ici.",
        },
        memory: {
          title: "Mémoire",
          description:
            "La RAM, le swap, la pression, l’historique et le classement des conteneurs apparaîtront ici.",
        },
        filesystems: {
          title: "Systèmes de fichiers",
          description:
            "Les montages, la capacité, les permissions, les E/S et l’historique apparaîtront ici.",
        },
        docker: {
          title: "Docker",
          description:
            "L’état, la santé, les ressources, les ports et les détails des conteneurs apparaîtront ici.",
        },
        events: {
          title: "Événements",
          description:
            "L’historique des avertissements et événements critiques apparaîtra ici.",
        },
        audit: {
          title: "Audit",
          description:
            "L’historique des actions administratives apparaîtra ici.",
        },
        backups: {
          title: "Sauvegardes",
          description:
            "L’état des sauvegardes, leur création manuelle et la restauration confirmée apparaîtront ici.",
        },
      },
      notFound: {
        eyebrow: "404",
        title: "Page introuvable",
        message: "La page demandée n’existe pas dans le tableau de bord.",
      },
      footer: {
        access: "Accès par tailnet uniquement",
        timezone: "Heures affichées en Africa/Tunis",
      },
    },
  },
} as const;
