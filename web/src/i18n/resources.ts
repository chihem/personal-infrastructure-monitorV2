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
      cpu: {
        eyebrow: "CPU monitoring",
        title: "Processor performance",
        introduction:
          "Inspect overall utilization, every detected logical vCPU, Linux load averages, and retained history.",
        refresh: "Refresh data",
        refreshing: "Refreshing",
        current: "Current measurement",
        overall: "Overall CPU",
        usage: "utilization",
        notAvailable: "N/A",
        notCollected: "No successful CPU measurement yet",
        observedAt: "Observed {{timestamp}}",
        currentErrorTitle: "Current CPU request failed",
        currentErrorMessage:
          "The latest CPU measurement could not be loaded. Existing history may still be available.",
        staleTitle: "CPU evidence is stale",
        staleMessage:
          "The last usable measurement was collected at {{timestamp}}. Values remain visible as last-known evidence.",
        unavailableTitle: "Current CPU is unavailable",
        unavailableMessage:
          "The collector has not produced a usable CPU measurement yet. No value has been estimated.",
        thresholds: {
          warning: "Warning at {{value}}%",
          critical: "Critical at {{value}}%",
          warningShort: "Warning 85%",
          criticalShort: "Critical 95%",
        },
        load: {
          subtitle: "Linux scheduler pressure",
          title: "Load averages",
          one: "1 minute",
          five: "5 minutes",
          fifteen: "15 minutes",
          help: "Load is the average number of runnable or waiting tasks. Compare it with the number of logical vCPUs; it is not a percentage.",
        },
        cores: {
          title: "Logical vCPU details",
          count: "{{count}} detected",
          logical: "Logical processor",
          unavailableTitle: "No vCPU details",
          unavailableMessage:
            "Logical processors will appear after the collector establishes a CPU baseline.",
        },
        metrics: {
          overall: "Overall CPU",
          core: "vCPU {{index}}",
          load1: "Load — 1 minute",
          load5: "Load — 5 minutes",
          load15: "Load — 15 minutes",
        },
        statistics: {
          minimum: "Minimum",
          average: "Average",
          peak: "Peak",
          gaps: "Gaps",
        },
        history: {
          eyebrow: "Retained evidence",
          title: "CPU history",
          retention: "14-day rolling retention",
          metric: "Measurement",
          range: "Time range",
          start: "Start",
          end: "End",
          apply: "Apply range",
          customTimezone: "Custom times use Africa/Tunis.",
          customInvalid:
            "Choose a valid past period with the end after the start and no more than 14 days.",
          customTitle: "Choose a custom period",
          customMessage:
            "Set a start and end within the retained 14-day history, then apply the range.",
          loadingTitle: "Loading CPU history",
          loadingMessage: "Reading the selected retained measurements.",
          errorTitle: "CPU history is unavailable",
          errorMessage:
            "The selected history could not be loaded. Try refreshing or choosing another valid range.",
          noObservedTitle: "No usable measurements",
          noObservedMessage:
            "This period contains no usable values. It includes {{gaps}} collection gaps.",
          noObservedSummary:
            "No observed values are available. Gaps and unavailable samples remain unfilled.",
          accessibleSummary:
            "Minimum {{minimum}}, average {{average}}, peak {{maximum}}. {{gaps}} gaps and {{unavailable}} unavailable buckets.",
          noMeasurement: "No measurement",
          showTable: "Show accessible data table",
          time: "Time",
          state: "Evidence state",
          period: "{{start}} to {{end}}",
          bucket: "{{seconds}}-second buckets",
          states: {
            observed: "Observed",
            unavailable: "Unavailable",
            gap: "Collection gap",
          },
        },
        ranges: {
          last_1m: "Last minute",
          last_5m: "Last 5 minutes",
          last_15m: "Last 15 minutes",
          last_30m: "Last 30 minutes",
          last_1h: "Last hour",
          last_6h: "Last 6 hours",
          last_24h: "Last 24 hours",
          last_7d: "Last 7 days",
          last_14d: "Last 14 days",
          custom: "Custom period",
        },
        reasons: {
          not_collected: "Not collected yet",
          not_supported: "Not supported",
          not_configured: "Not configured",
          collector_error: "Collector error",
          permission_denied: "Permission denied",
          dependency_unavailable: "Dependency unavailable",
        },
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
      cpu: {
        eyebrow: "Surveillance du processeur",
        title: "Performances du processeur",
        introduction:
          "Consultez l’utilisation globale, chaque vCPU logique détecté, les charges Linux et l’historique conservé.",
        refresh: "Actualiser les données",
        refreshing: "Actualisation",
        current: "Mesure actuelle",
        overall: "Processeur global",
        usage: "utilisation",
        notAvailable: "N/D",
        notCollected: "Aucune mesure CPU réussie pour le moment",
        observedAt: "Observé le {{timestamp}}",
        currentErrorTitle: "Échec de la demande CPU actuelle",
        currentErrorMessage:
          "La dernière mesure CPU n’a pas pu être chargée. L’historique existant peut rester disponible.",
        staleTitle: "Les données CPU sont anciennes",
        staleMessage:
          "La dernière mesure exploitable date du {{timestamp}}. Les valeurs restent visibles comme dernière preuve connue.",
        unavailableTitle: "Le CPU actuel est indisponible",
        unavailableMessage:
          "Le collecteur n’a pas encore produit de mesure CPU exploitable. Aucune valeur n’a été estimée.",
        thresholds: {
          warning: "Avertissement à {{value}} %",
          critical: "Critique à {{value}} %",
          warningShort: "Avertissement 85 %",
          criticalShort: "Critique 95 %",
        },
        load: {
          subtitle: "Pression de l’ordonnanceur Linux",
          title: "Charges moyennes",
          one: "1 minute",
          five: "5 minutes",
          fifteen: "15 minutes",
          help: "La charge est le nombre moyen de tâches exécutables ou en attente. Comparez-la au nombre de vCPU logiques ; ce n’est pas un pourcentage.",
        },
        cores: {
          title: "Détails des vCPU logiques",
          count: "{{count}} détectés",
          logical: "Processeur logique",
          unavailableTitle: "Aucun détail vCPU",
          unavailableMessage:
            "Les processeurs logiques apparaîtront lorsque le collecteur aura établi sa référence CPU.",
        },
        metrics: {
          overall: "Processeur global",
          core: "vCPU {{index}}",
          load1: "Charge — 1 minute",
          load5: "Charge — 5 minutes",
          load15: "Charge — 15 minutes",
        },
        statistics: {
          minimum: "Minimum",
          average: "Moyenne",
          peak: "Pic",
          gaps: "Lacunes",
        },
        history: {
          eyebrow: "Données conservées",
          title: "Historique CPU",
          retention: "Conservation glissante de 14 jours",
          metric: "Mesure",
          range: "Période",
          start: "Début",
          end: "Fin",
          apply: "Appliquer la période",
          customTimezone: "Les heures personnalisées utilisent Africa/Tunis.",
          customInvalid:
            "Choisissez une période passée valide, avec une fin postérieure au début et une durée maximale de 14 jours.",
          customTitle: "Choisir une période personnalisée",
          customMessage:
            "Définissez un début et une fin dans les 14 jours conservés, puis appliquez la période.",
          loadingTitle: "Chargement de l’historique CPU",
          loadingMessage: "Lecture des mesures conservées sélectionnées.",
          errorTitle: "L’historique CPU est indisponible",
          errorMessage:
            "L’historique sélectionné n’a pas pu être chargé. Actualisez ou choisissez une autre période valide.",
          noObservedTitle: "Aucune mesure exploitable",
          noObservedMessage:
            "Cette période ne contient aucune valeur exploitable. Elle comprend {{gaps}} lacunes de collecte.",
          noObservedSummary:
            "Aucune valeur observée n’est disponible. Les lacunes et mesures indisponibles ne sont pas remplacées.",
          accessibleSummary:
            "Minimum {{minimum}}, moyenne {{average}}, pic {{maximum}}. {{gaps}} lacunes et {{unavailable}} intervalles indisponibles.",
          noMeasurement: "Aucune mesure",
          showTable: "Afficher le tableau de données accessible",
          time: "Heure",
          state: "État de la preuve",
          period: "Du {{start}} au {{end}}",
          bucket: "Intervalles de {{seconds}} secondes",
          states: {
            observed: "Observé",
            unavailable: "Indisponible",
            gap: "Lacune de collecte",
          },
        },
        ranges: {
          last_1m: "Dernière minute",
          last_5m: "5 dernières minutes",
          last_15m: "15 dernières minutes",
          last_30m: "30 dernières minutes",
          last_1h: "Dernière heure",
          last_6h: "6 dernières heures",
          last_24h: "24 dernières heures",
          last_7d: "7 derniers jours",
          last_14d: "14 derniers jours",
          custom: "Période personnalisée",
        },
        reasons: {
          not_collected: "Pas encore collecté",
          not_supported: "Non pris en charge",
          not_configured: "Non configuré",
          collector_error: "Erreur du collecteur",
          permission_denied: "Permission refusée",
          dependency_unavailable: "Dépendance indisponible",
        },
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
